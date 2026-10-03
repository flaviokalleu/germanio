package runtime

import (
	"testing"
	"time"
)

// GEP 0027 num domínio sem GitLab: revisões de um livro (origem → destino)
// mescladas de três formas, juntando commits, e mescladas quando a compilação
// da origem passa.

type livro struct {
	t    *testing.T
	c    *client
	base string
}

func (l *livro) put(branch, path, content string) {
	l.t.Helper()
	l.c.expect("PUT", l.base+"/repositorio/arquivos/"+path, map[string]any{"branch": branch, "conteudo": content, "mensagem": "escreve " + path}, 200)
}

func (l *livro) branch(name, from string) {
	l.t.Helper()
	l.c.expect("POST", l.base+"/repositorio/branches", map[string]any{"nome": name, "origem": from}, 201)
}

func (l *livro) commits(ref string) []map[string]any {
	l.t.Helper()
	_, _, raw := l.c.do("GET", l.base+"/repositorio/commits?ref_name="+ref, nil)
	return decodeList(l.t, raw)
}

func (l *livro) review(title, source string, extra map[string]any) string {
	l.t.Helper()
	body := map[string]any{"titulo": title, "origem": source, "destino": "main"}
	for k, v := range extra {
		body[k] = v
	}
	r := l.c.expect("POST", l.base+"/revisoes", body, 201)
	return l.base + "/revisoes/" + itoa(int(r["numero"].(float64)))
}

func parents(c map[string]any) int { return len(c["parent_ids"].([]any)) }

func livros(t *testing.T) *livro {
	_, c := loadApp(t, "testdata/intencao/livros.ge")
	c.expect("POST", "/cadastro", map[string]any{"nome": "Ana", "email": "ana@x.com", "senha": "senha-da-ana"}, 201)
	c.csrf = csrfFromCookie(t, c)
	l := c.expect("POST", "/_ge/api/livros", map[string]any{"nome": "atlas"}, 201)
	return &livro{t: t, c: c, base: "/_ge/api/livros/" + itoa(int(l["id"].(float64)))}
}

func TestFormasDeMesclar(t *testing.T) {
	l := livros(t)
	l.put("main", "cap1.txt", "um\n")
	l.branch("rev1", "main")
	l.branch("rev2", "main")
	l.branch("rev3", "main")
	l.put("rev1", "cap2.txt", "dois\n")
	l.put("rev1", "cap3.txt", "três\n")
	l.put("rev2", "cap4.txt", "quatro\n")
	l.put("rev3", "cap5.txt", "cinco\n")
	l.put("main", "prefacio.txt", "prefácio\n")

	// juntar commits: um commit sobre o destino, com o título da revisão
	r1 := l.review("Capítulos 2 e 3", "rev1", map[string]any{"juntar_commits": true})
	if got := l.c.expect("GET", r1, nil, 200); got["juntar_commits"] != true {
		t.Fatalf("juntar_commits no registro: %v", got)
	}
	m := l.c.expect("POST", r1+"/mesclar", nil, 200)
	head := l.commits("main")[0]
	if m["estado"] != "mesclada" || m["commit_mesclagem"] != head["id"] || parents(head) != 1 || head["title"] != "Capítulos 2 e 3" {
		t.Fatalf("juntar commits: %v / %v", m, head)
	}
	if ch := l.c.expect("GET", r1+"/mudancas", nil, 200); len(ch["mudancas"].([]any)) != 2 {
		t.Fatalf("mudanças depois de juntar: %v", ch["mudancas"])
	}

	// forma de mesclar do livro: valor fora da lista é recusado
	l.c.expect("PATCH", l.base, map[string]any{"forma_de_mesclar": "qualquer"}, 400)
	l.c.expect("PATCH", l.base, map[string]any{"forma_de_mesclar": "linear"}, 200)

	// linear: a origem é posta em dia e o destino só avança; nenhum commit de mescla
	before := l.commits("main")[0]["id"]
	r2 := l.review("Capítulo 4", "rev2", nil)
	l.c.expect("POST", r2+"/mesclar", nil, 200)
	main := l.commits("main")
	if parents(main[0]) != 1 || main[0]["title"] != "escreve cap4.txt" || main[1]["id"] != before {
		t.Fatalf("histórico linear: %v", main[:2])
	}
	if src := l.commits("rev2")[0]; src["id"] != main[0]["id"] {
		t.Fatalf("a origem deveria estar posta em dia: %v", src)
	}
	if ch := l.c.expect("GET", r2+"/mudancas", nil, 200); len(ch["mudancas"].([]any)) != 1 {
		t.Fatalf("mudanças depois do avanço: %v", ch["mudancas"])
	}

	// semi_linear: commit de mescla sobre a origem posta em dia
	l.c.expect("PATCH", l.base, map[string]any{"forma_de_mesclar": "semi_linear"}, 200)
	before = l.commits("main")[0]["id"]
	r3 := l.review("Capítulo 5", "rev3", nil)
	l.c.expect("POST", r3+"/mesclar", nil, 200)
	head = l.commits("main")[0]
	if ps := head["parent_ids"].([]any); len(ps) != 2 || ps[0] != before {
		t.Fatalf("semi linear: %v", head)
	}
	if src := l.commits("rev3"); src[1]["id"] != before {
		t.Fatalf("a origem deveria estar sobre o destino: %v", src[:2])
	}
}

func waitReview(t *testing.T, c *client, path string, until func(map[string]any) bool) map[string]any {
	t.Helper()
	var got map[string]any
	for i := 0; i < 150; i++ {
		got = c.expect("GET", path, nil, 200)
		if until(got) {
			return got
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("a revisão não chegou ao estado esperado: %v", got)
	return nil
}

// runsDone: no execution of the book is still waiting or running.
func runsDone(t *testing.T, l *livro) bool {
	_, _, raw := l.c.do("GET", l.base+"/compilacoes", nil)
	for _, r := range decodeList(t, raw) {
		if st := r["estado"]; st == "pendente" || st == "executando" || st == "criado" {
			return false
		}
	}
	return true
}

func TestMesclarQuandoPassar(t *testing.T) {
	t.Setenv("GERMANIO_EXECUTOR", "local")
	l := livros(t)
	l.put("main", "compilar.yml", "estagios: [testar]\netapas:\n  testar:\n    estagio: testar\n    comandos: [\"sleep 2\", \"test -f ok.txt\"]\n")

	// a compilação da origem passa: a revisão é mesclada por quem pediu
	l.branch("boa", "main")
	l.put("boa", "ok.txt", "ok\n")
	r1 := l.review("Boa", "boa", nil)
	s := l.c.expect("POST", r1+"/mesclar", map[string]any{"mesclar_quando_passar": true}, 200)
	if s["estado"] != "aberta" || s["mesclar_quando_passar"] != true {
		t.Fatalf("mesclagem agendada: %v", s)
	}
	done := waitReview(t, l.c, r1, func(m map[string]any) bool { return m["estado"] != "aberta" })
	if done["estado"] != "mesclada" || done["mesclar_quando_passar"] != false || done["commit_mesclagem"] == nil {
		t.Fatalf("depois da compilação: %v", done)
	}

	// a compilação falha: a revisão para de esperar e continua aberta
	l.branch("ruim", "main")
	l.put("ruim", "compilar.yml", "etapas:\n  testar:\n    comandos: [\"sleep 1\", \"exit 1\"]\n")
	r2 := l.review("Ruim", "ruim", nil)
	l.c.expect("POST", r2+"/mesclar", map[string]any{"mesclar_quando_passar": true}, 200)
	stopped := waitReview(t, l.c, r2, func(m map[string]any) bool { return m["mesclar_quando_passar"] == false })
	if stopped["estado"] != "aberta" {
		t.Fatalf("falha não pode mesclar: %v", stopped)
	}
	// pedir de novo, com a última compilação falha, é recusado e explicado
	_, out, _ := l.c.do("POST", r2+"/mesclar", map[string]any{"mesclar_quando_passar": true})
	if msg, _ := out["message"].(string); msg == "" {
		t.Fatalf("recusa sem explicação: %v", out)
	}

	// cancelar a espera: a compilação passa e nada é mesclado
	l.branch("espera", "main")
	l.put("espera", "ok.txt", "ok\n")
	r3 := l.review("Espera", "espera", nil)
	l.c.expect("POST", r3+"/mesclar", map[string]any{"mesclar_quando_passar": true}, 200)
	if c := l.c.expect("POST", r3+"/cancelar_mesclagem", nil, 200); c["mesclar_quando_passar"] != false {
		t.Fatalf("cancelar: %v", c)
	}
	for i := 0; i < 150 && !runsDone(t, l); i++ {
		time.Sleep(100 * time.Millisecond)
	}
	if got := l.c.expect("GET", r3, nil, 200); got["estado"] != "aberta" {
		t.Fatalf("mesclada depois de cancelar: %v", got)
	}

	// sem compilação em andamento (já passou), mescla na hora
	got := l.c.expect("POST", r3+"/mesclar", map[string]any{"mesclar_quando_passar": true}, 200)
	if got["estado"] != "mesclada" {
		t.Fatalf("compilação já passou: %v", got)
	}
}
