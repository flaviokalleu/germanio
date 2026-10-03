package runtime

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flaviokalleu/germanio/runtime/git"
)

// editora starts the publishing house (a domain without GitLab) with ana
// (editor), bia (leitor), caio (autor) and eve (no role).
func editora(t *testing.T) (*App, map[string]*client, string) {
	t.Helper()
	t.Setenv("GERMANIO_ARQUIVOS", filepath.Join(t.TempDir(), "arquivos"))
	t.Setenv("GERMANIO_LFS_MAX_MB", "1")
	app, ana := loadApp(t, "testdata/intencao/editora.ge")
	people := map[string]*client{"ana": ana}
	ids := map[string]float64{}
	for _, n := range []string{"ana", "bia", "caio", "eve"} {
		c := ana
		if n != "ana" {
			_, c = ana.fresh(t)
			people[n] = c
		}
		me := c.expect("POST", "/cadastro", map[string]any{"nome": n, "email": n + "@x.com", "senha": "senha-" + n + "-1"}, 201)
		c.csrf = csrfFromCookie(t, c)
		ids[n] = me["id"].(float64)
	}
	ana.expect("POST", "/_ge/api/livros", map[string]any{"titulo": "Contos", "caminho": "contos"}, 201)
	ana.expect("POST", "/_ge/api/livros/1/membros", map[string]any{"pessoa_id": ids["bia"], "papel": "leitor"}, 201)
	ana.expect("POST", "/_ge/api/livros/1/membros", map[string]any{"pessoa_id": ids["caio"], "papel": "autor"}, 201)
	return app, people, ana.base
}

type lfsReply struct {
	status int
	header http.Header
	body   []byte
}

func lfsDo(t *testing.T, method, url, user string, header map[string]string, body []byte) lfsReply {
	t.Helper()
	req, _ := http.NewRequest(method, url, bytes.NewReader(body))
	req.Header.Set("Accept", git.LFSMediaType)
	if method == "POST" {
		req.Header.Set("Content-Type", git.LFSMediaType)
	}
	if user != "" {
		req.SetBasicAuth(user+"@x.com", "senha-"+user+"-1")
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return lfsReply{resp.StatusCode, resp.Header, b}
}

func batch(t *testing.T, base, user, op string, objs ...git.LFSObject) (int, git.LFSBatchResponse) {
	t.Helper()
	body, _ := json.Marshal(git.LFSBatchRequest{Operation: op, Transfers: []string{"basic"}, Objects: objs, HashAlgo: "sha256"})
	r := lfsDo(t, "POST", base+"/contos.git/info/lfs/objects/batch", user, nil, body)
	var out git.LFSBatchResponse
	json.Unmarshal(r.body, &out)
	if r.status == 200 && r.header.Get("Content-Type") != git.LFSMediaType {
		t.Fatalf("tipo da resposta do batch: %q", r.header.Get("Content-Type"))
	}
	return r.status, out
}

func sha(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// GEP 0035: the batch API with the rules of clone and push, uploads checked
// by SHA-256 and size, the size limit, and transfer links that only serve
// their own object and direction.
func TestGitLFS(t *testing.T) {
	_, p, base := editora(t)
	content := []byte("capítulo 1 — um arquivo grande de verdade")
	obj := git.LFSObject{OID: sha(content), Size: int64(len(content))}

	// who may not
	if st, _ := batch(t, base, "", "download", obj); st != 401 {
		t.Fatalf("anônimo: %d", st)
	}
	r := lfsDo(t, "POST", base+"/contos.git/info/lfs/objects/batch", "", nil, []byte(`{"operation":"download","objects":[]}`))
	if r.header.Get("LFS-Authenticate") == "" {
		t.Fatal("401 sem LFS-Authenticate: o cliente não pediria credenciais")
	}
	if st, _ := batch(t, base, "eve", "download", obj); st != 404 {
		t.Fatalf("quem não vê o livro: %d", st)
	}
	if st, _ := batch(t, base, "bia", "upload", obj); st != 403 {
		t.Fatalf("leitor enviando: %d", st)
	}

	// upload by the author
	st, up := batch(t, base, "caio", "upload", obj, git.LFSObject{OID: sha([]byte("x")), Size: 2 << 20})
	if st != 200 || len(up.Objects) != 2 {
		t.Fatalf("batch de envio: %d %+v", st, up)
	}
	act := up.Objects[0].Actions["upload"]
	if act.Href != base+"/contos.git/info/lfs/objects/"+obj.OID || act.Header["Authorization"] == "" {
		t.Fatalf("ação de envio: %+v", act)
	}
	if e := up.Objects[1].Error; e == nil || e.Code != 422 {
		t.Fatalf("objeto acima do limite deveria ter erro 422: %+v", up.Objects[1])
	}
	if r := lfsDo(t, "PUT", act.Href, "", act.Header, []byte("conteúdo trocado por alguém no caminho!!!!")); r.status != 422 {
		t.Fatalf("conteúdo com outro sha256: %d %s", r.status, r.body)
	}
	if r := lfsDo(t, "PUT", base+"/contos.git/info/lfs/objects/"+sha([]byte("outro")), "", act.Header, []byte("outro")); r.status != 401 {
		t.Fatalf("link de um objeto usado para outro: %d", r.status)
	}
	if r := lfsDo(t, "GET", act.Href, "", act.Header, nil); r.status != 401 {
		t.Fatalf("link de envio usado para baixar: %d", r.status)
	}
	if r := lfsDo(t, "PUT", act.Href, "", act.Header, content); r.status != 200 {
		t.Fatalf("envio: %d %s", r.status, r.body)
	}
	// limit also without the batch (basic auth straight to the object)
	big := bytes.Repeat([]byte("a"), 1<<20+1)
	if r := lfsDo(t, "PUT", base+"/contos.git/info/lfs/objects/"+sha(big), "caio", nil, big); r.status != 413 {
		t.Fatalf("objeto acima do limite: %d", r.status)
	}
	if r := lfsDo(t, "PUT", base+"/contos.git/info/lfs/objects/"+sha([]byte("y")), "bia", nil, []byte("y")); r.status != 403 {
		t.Fatalf("leitor enviando objeto direto: %d", r.status)
	}

	// already there: no upload action
	if _, again := batch(t, base, "caio", "upload", obj); again.Objects[0].Actions != nil {
		t.Fatalf("objeto existente pedido de novo: %+v", again.Objects[0])
	}
	// download by the reader
	st, down := batch(t, base, "bia", "download", obj, git.LFSObject{OID: sha([]byte("nada")), Size: 4})
	if st != 200 || down.Objects[1].Error == nil || down.Objects[1].Error.Code != 404 {
		t.Fatalf("batch de download: %d %+v", st, down)
	}
	da := down.Objects[0].Actions["download"]
	if r := lfsDo(t, "GET", da.Href, "", da.Header, nil); r.status != 200 || !bytes.Equal(r.body, content) {
		t.Fatalf("download: %d %q", r.status, r.body)
	}
	if r := lfsDo(t, "GET", da.Href, "bia", nil, nil); r.status != 200 || !bytes.Equal(r.body, content) {
		t.Fatalf("download com as credenciais do git: %d", r.status)
	}
	if r := lfsDo(t, "GET", da.Href, "eve", nil, nil); r.status != 404 {
		t.Fatalf("download por quem não vê o livro: %d", r.status)
	}
	if r := lfsDo(t, "GET", da.Href, "", map[string]string{"Authorization": "Germanio-LFS forjado.abc"}, nil); r.status != 401 {
		t.Fatalf("link forjado: %d", r.status)
	}

	// a read-only record receives nothing, not even large files
	p["ana"].expect("PATCH", "/_ge/api/livros/1", map[string]any{"arquivado": true}, 200)
	if st, _ := batch(t, base, "caio", "upload", git.LFSObject{OID: sha([]byte("z")), Size: 1}); st != 403 {
		t.Fatalf("livro arquivado recebendo arquivos: %d", st)
	}
	if st, _ := batch(t, base, "bia", "download", obj); st != 200 {
		t.Fatalf("livro arquivado continua legível: %d", st)
	}

	// removing the record removes its objects
	p["ana"].expect("PATCH", "/_ge/api/livros/1", map[string]any{"arquivado": false}, 200)
	p["ana"].expect("DELETE", "/_ge/api/livros/1", nil, 204)
	var left []string
	filepath.WalkDir(filepath.Join(os.Getenv("GERMANIO_ARQUIVOS"), "lfs"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			left = append(left, p)
		}
		return nil
	})
	if len(left) != 0 {
		t.Fatalf("objetos ficaram depois de excluir o livro: %v", left)
	}
}

// gitServer serves bare repositories of dir with git's own http-backend,
// behind basic authentication: another Git server, on this machine.
func gitServer(t *testing.T, dir, user, pass string) string {
	t.Helper()
	out, err := exec.Command("git", "--exec-path").Output()
	if err != nil {
		t.Skip("git indisponível")
	}
	h := &cgi.Handler{Path: filepath.Join(strings.TrimSpace(string(out)), "git-http-backend"), Env: []string{"GIT_PROJECT_ROOT=" + dir, "GIT_HTTP_EXPORT_ALL=1", "REMOTE_USER=x"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); !ok || u != user || p != pass {
			w.Header().Set("WWW-Authenticate", `Basic realm="x"`)
			http.Error(w, "auth", 401)
			return
		}
		h.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for end := time.Now().Add(20 * time.Second); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		if ok() {
			return
		}
	}
	t.Fatalf("esperando: %s", what)
}

// GEP 0036: mirrors receive the code after every change, or bring it
// periodically; credentials stay hidden; failures are recorded without
// breaking the change; local addresses need GERMANIO_PERMITIR_REDE_LOCAL.
func TestEspelhos(t *testing.T) {
	_, p, _ := editora(t)
	ana := p["ana"]
	remoteDir := t.TempDir()
	remote, err := git.NewStore(remoteDir)
	if err != nil {
		t.Fatal(err)
	}
	remote.Init("destino.git", "main")
	exec.Command("git", "--git-dir", filepath.Join(remoteDir, "destino.git"), "config", "http.receivepack", "true").Run()
	srv := gitServer(t, remoteDir, "ana", "s3gr3do-do-espelho")
	u := strings.Replace(srv, "http://", "http://ana:s3gr3do-do-espelho@", 1) + "/destino.git"

	// without the permission, a local address is refused with the reason
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "")
	if code, _, raw := ana.do("POST", "/_ge/api/livros/1/copias", map[string]any{"url": u}); code != 400 || !strings.Contains(raw, "GERMANIO_PERMITIR_REDE_LOCAL") {
		t.Fatalf("endereço local sem permissão: %d %s", code, raw)
	}
	for _, bad := range []string{"file:///etc/passwd", "ext::sh -c id", "ssh://git@exemplo.com/x.git"} {
		if code, _, _ := ana.do("POST", "/_ge/api/livros/1/copias", map[string]any{"url": bad}); code != 400 {
			t.Fatalf("%s aceito: %d", bad, code)
		}
	}
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1")
	if code, _, _ := p["caio"].do("POST", "/_ge/api/livros/1/copias", map[string]any{"url": u}); code != 403 {
		t.Fatalf("autor criando espelho: %d", code)
	}
	_, _, raw := ana.do("POST", "/_ge/api/livros/1/copias", map[string]any{"url": u})
	if strings.Contains(raw, "s3gr3do") || strings.Contains(raw, "credencial") {
		t.Fatalf("a credencial voltou na resposta: %s", raw)
	}
	mirror := func(id string) map[string]any { return ana.expect("GET", "/_ge/api/livros/1/copias/"+id, nil, 200) }
	if m := mirror("1"); m["url"] != srv+"/destino.git" || m["sentido"] != "enviar" {
		t.Fatalf("espelho criado: %v", m)
	}
	if _, _, list := ana.do("GET", "/_ge/api/livros/1/copias", nil); strings.Contains(list, "s3gr3do") {
		t.Fatalf("a credencial aparece na lista: %s", list)
	}

	// a change of the code goes to the mirror
	edit := p["caio"].expect("PUT", "/_ge/api/livros/1/repositorio/arquivos/capitulo1.md", map[string]any{"conteudo": "Era uma vez", "branch": "main"}, 200)
	waitFor(t, "o código chegar ao espelho", func() bool {
		got, _ := remote.Resolve("destino.git", "refs/heads/main")
		return got == edit["commit_id"] && mirror("1")["situacao"] == "atualizada"
	})
	if m := mirror("1"); m["ultimo_sucesso"] == nil || (m["ultimo_erro"] != nil && m["ultimo_erro"] != "") {
		t.Fatalf("registro da atualização: %v", m)
	}

	// a failing mirror is recorded, never breaks the change, never shows the password
	bad := strings.Replace(u, "s3gr3do-do-espelho", "senha-errada-xyz", 1)
	ana.expect("POST", "/_ge/api/livros/1/copias", map[string]any{"url": bad}, 201)
	p["caio"].expect("PUT", "/_ge/api/livros/1/repositorio/arquivos/capitulo2.md", map[string]any{"conteudo": "Fim", "branch": "main"}, 200)
	waitFor(t, "a falha ficar registrada", func() bool {
		m := mirror("2")
		return m["situacao"] == "falhou" && toStrT(m["ultimo_erro"]) != ""
	})
	if e := mirror("2")["ultimo_erro"].(string); strings.Contains(e, "senha-errada-xyz") {
		t.Fatalf("a senha aparece no erro: %s", e)
	}
	// changing the address forgets the credentials (they never follow to another server)
	ana.expect("PATCH", "/_ge/api/livros/1/copias/1", map[string]any{"url": strings.Replace(srv, "127.0.0.1", "localhost", 1) + "/destino.git"}, 200)
	waitFor(t, "o espelho sem credencial falhar", func() bool { return mirror("1")["situacao"] == "falhou" })
	ana.expect("PATCH", "/_ge/api/livros/1/copias/1", map[string]any{"url": srv + "/destino.git"}, 200)
	waitFor(t, "voltar ao endereço antigo não traz a credencial de volta", func() bool { return mirror("1")["situacao"] == "falhou" })

	// a receiving mirror makes another book a copy of the remote
	ana.expect("POST", "/_ge/api/livros", map[string]any{"titulo": "Cópia", "caminho": "copia"}, 201)
	ana.expect("POST", "/_ge/api/livros/2/copias", map[string]any{"url": u, "sentido": "receber"}, 201)
	want, _ := remote.Resolve("destino.git", "refs/heads/main")
	waitFor(t, "o livro receber o código do espelho", func() bool {
		code, _, raw := ana.do("GET", "/_ge/api/livros/2/repositorio/branches", nil)
		return code == 200 && strings.Contains(raw, want)
	})
}
