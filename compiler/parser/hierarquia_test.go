package parser

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/lexer"
)

// resolved parses and resolves an intent program.
func resolved(t *testing.T, src string) *ast.App {
	t.Helper()
	toks, err := lexer.New(src).Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	prog, err := New(toks).Parse()
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, src)
	}
	if err := ResolveIntent(prog); err != nil {
		t.Fatalf("resolve: %v\n%s", err, src)
	}
	return prog.App
}

// canonical drops positions (they differ between the two forms) and keeps
// every fact, so equal canonical forms mean the same application.
func canonical(t *testing.T, app *ast.App) any {
	t.Helper()
	b, err := json.Marshal(app)
	if err != nil {
		t.Fatal(err)
	}
	var v any
	json.Unmarshal(b, &v)
	var strip func(any) any
	strip = func(x any) any {
		switch m := x.(type) {
		case map[string]any:
			delete(m, "Pos")
			for k, val := range m {
				m[k] = strip(val)
			}
		case []any:
			for i := range m {
				m[i] = strip(m[i])
			}
		}
		return x
	}
	return strip(v)
}

const base = `crie sistema x

tenha usuarios, grupos, membros

cada usuario tem
    email obrigatório e único
    senha

tenha login

tenha papeis
    guest 10
    reporter 20
    developer 30
    maintainer 40
    owner 50

grupo tem
    nome obrigatório
    visibilidade
    membros com papel
`

// Cada linha da tabela de seções (docs/INTENCAO.md) é igual à frase plana.
func TestHierarquiaEquivaleAFrasePlana(t *testing.T) {
	cases := []struct{ name, flat, tree string }{
		{"tem", "tenha projetos\n\ncada projeto tem\n    nome obrigatório\n    descrição\n",
			"projetos\n    tem\n        nome obrigatório\n        descrição\n"},
		{"pertence a", "tenha projetos\n\ncada projeto tem\n    nome\n\nprojeto pertence a grupo opcional\n",
			"projetos\n    tem\n        nome\n    pertence a\n        grupo opcional\n"},
		{"pertence a (na linha)", "tenha projetos\n\ncada projeto tem\n    nome\n\nprojeto pertence a grupo opcional\n",
			"projetos\n    tem\n        nome\n    pertence a grupo opcional\n"},
		{"começa e pode", "tenha projetos\n\ncada projeto tem\n    nome\n\nprojeto começa aberto\n\nprojeto pode\n    fechar\n    reabrir\n    ser arquivado\n",
			"projetos\n    tem\n        nome\n    começa aberto\n    pode\n        fechar\n        reabrir\n        ser arquivado\n"},
		{"regras", "tenha projetos\n\ncada projeto tem\n    nome\n    visibilidade\n    membros com papel\n\nprojeto pertence a grupo opcional\nprojeto herda membros do grupo\nquem cria projeto vira owner\ntodo projeto precisa ter pelo menos um owner\nprojeto não pode ser mais visível que o grupo\nprojeto pode ser arquivado\nprojeto arquivado é somente leitura\n",
			"projetos\n    tem\n        nome\n        visibilidade\n        membros com papel\n    pertence a grupo opcional\n    pode\n        ser arquivado\n    regras\n        herda membros do grupo\n        quem cria vira owner\n        precisa de pelo menos um owner\n        não pode ser mais visível que o grupo\n        arquivado é somente leitura\n"},
		{"acesso", "tenha projetos\n\ncada projeto tem\n    nome\n    membros com papel\n    repositório\n    caminho único\n\nguest pode ver projetos\nreporter pode baixar código dos projetos\ndeveloper pode\n    criar projetos\n    enviar código para projetos\nsomente maintainer pode enviar código para a branch padrão dos projetos\nsomente owner pode excluir projetos\nusuario pode criar seus projetos\nmaintainer pode adicionar membros\n",
			"projetos\n    tem\n        nome\n        membros com papel\n        repositório\n        caminho único\n    acesso\n        guest\n            ver\n        reporter\n            baixar código\n        developer\n            criar\n            enviar código\n        somente maintainer\n            enviar código para a branch padrão\n        somente owner\n            excluir\n        usuario\n            criar seus\n        maintainer\n            adicionar membros\n"},
		{"permita e integração", "tenha projetos\n\ncada projeto tem\n    nome\n    cidade\n\npermita pesquisar projetos\npermita filtrar projetos por cidade\n\ndisponibilize projetos para integração como \"projects\"\n",
			"projetos\n    tem\n        nome\n        cidade\n    permita\n        pesquisar\n        filtrar por cidade\n    integração\n        nome \"projects\"\n"},
		{"estados de visibilidade", "tenha issues\n\ncada issue tem\n    titulo\n    autor\n    responsaveis\n\nissue começa aberta\nissue pode ser confidencial\nissue confidencial pode ser vista por\n    autor\n    responsaveis\n    reporter ou superior\nissue fechada é final\nissue pode fechar\n",
			"issues\n    tem\n        titulo\n        autor\n        responsaveis\n    começa aberta\n    pode\n        ser confidencial\n        fechar\n    regras\n        confidencial pode ser vista por\n            autor\n            responsaveis\n            reporter ou superior\n        fechada é final\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			flat := canonical(t, resolved(t, base+"\n"+c.flat))
			tree := canonical(t, resolved(t, base+"\n"+c.tree))
			if !reflect.DeepEqual(flat, tree) {
				fb, _ := json.MarshalIndent(flat, "", " ")
				tb, _ := json.MarshalIndent(tree, "", " ")
				t.Fatalf("formas diferentes\nplana:\n%s\nhierárquica:\n%s", diffHint(string(fb), string(tb)), "")
				_ = tb
			}
		})
	}
}

// diffHint shows the first lines where two dumps differ.
func diffHint(a, b string) string {
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < len(al) && i < len(bl); i++ {
		if al[i] != bl[i] {
			lo := max(0, i-6)
			return strings.Join(al[lo:min(len(al), i+4)], "\n") + "\n--- versus ---\n" + strings.Join(bl[lo:min(len(bl), i+4)], "\n")
		}
	}
	return "(tamanhos diferentes)"
}

// A comparação distingue aplicativos diferentes (o teste acima não passa por acaso).
func TestHierarquiaComparacaoDetectaDiferenca(t *testing.T) {
	a := canonical(t, resolved(t, base+"\ntenha projetos\n\ncada projeto tem\n    nome\n\nguest pode ver projetos\n"))
	b := canonical(t, resolved(t, base+"\nprojetos\n    tem\n        nome\n    acesso\n        reporter\n            ver\n"))
	if reflect.DeepEqual(a, b) {
		t.Fatal("guest e reporter deveriam dar aplicativos diferentes")
	}
}

func TestHierarquiaDemaisSecoes(t *testing.T) {
	cases := []struct{ name, flat, tree string }{
		{"quando e antes de", "tenha projetos\n\ncada projeto tem\n    nome\n\nantes de criar projeto\n    recuse \"não\"\n\nquando excluir projeto\n    mostrar \"ok\"\n",
			"projetos\n    tem\n        nome\n    antes de criar\n        recuse \"não\"\n    quando excluir\n        mostrar \"ok\"\n"},
		{"página", "tenha projetos\n\ncada projeto tem\n    nome\n\ncrie página Projetos\n    mostre projetos\n    permita\n        pesquisar\n        criar\n",
			"projetos\n    tem\n        nome\n\npágina Projetos\n    mostre projetos\n    permita\n        pesquisar\n        criar\n"},
		{"recebe e repositório", "tenha merge requests\n\ncada merge request tem\n    titulo\n\nmerge request recebe aprovações\n\ntenha projetos\n\ncada projeto tem\n    nome\n    caminho único\n    repositório\n\nrepositório do projeto pode começar com \"README.md\" contendo \"# {nome}\"\n",
			"merge requests\n    tem\n        titulo\n    recebe aprovações\n\nprojetos\n    tem\n        nome\n        caminho único\n        repositório\n    repositório pode começar com \"README.md\" contendo \"# {nome}\"\n"},
		{"um item na mesma linha", "tenha projetos\n\ncada projeto tem\n    caminho único\n\nprojeto tem repositório\nprojeto começa aberto\nprojeto pode fechar\n",
			"projetos\n    tem\n        caminho único\n    tem repositório\n    começa aberto\n    pode fechar\n"},
		{"pendência para", "tenha chamados\n\ncada chamado tem\n    titulo\n    responsaveis\n\nchamado gera pendência para responsaveis\n",
			"chamados\n    tem\n        titulo\n        responsaveis\n    pendência para\n        responsaveis\n"},
		{"guarda histórico", "tenha issues\n\ncada issue tem\n    titulo\n\nissue guarda histórico\n",
			"issues\n    tem\n        titulo\n    guarda histórico\n"},
		{"usam variaveis", "projetos\n    tem\n        caminho único\n        pipelines\n    tem repositório\n    executa pipelines a cada envio de código conforme \"ci.yml\"\n\npipelines\n    tem\n        jobs\n\njobs\n    tem\n        nome\n\nvariaveis\n    tem\n        chave\n        valor texto oculto\n    pertence a projeto\n\npipelines usam as variaveis do projeto\n",
			"projetos\n    tem\n        caminho único\n        pipelines\n    tem repositório\n    executa pipelines a cada envio de código conforme \"ci.yml\"\n\npipelines\n    tem\n        jobs\n    usam variaveis do projeto\n\njobs\n    tem\n        nome\n\nvariaveis\n    tem\n        chave\n        valor texto oculto\n    pertence a projeto\n"},
		{"branches protegidas", "projetos\n    tem\n        caminho único\n        membros com papel\n    tem repositório\n\nbranches protegidas\n    singular branch protegida\n    tem\n        nome\n    pertence a projeto\n\nsomente maintainer pode enviar código para as branches protegidas dos projetos\n",
			"projetos\n    tem\n        caminho único\n        membros com papel\n    tem repositório\n    acesso\n        somente maintainer\n            enviar código para as branches protegidas\n\nbranches protegidas\n    singular branch protegida\n    tem\n        nome\n    pertence a projeto\n"},
		{"dado chamado mensagens", "tenha mensagens\n\ncada mensagem tem\n    texto\n",
			"mensagens\n    tem\n        texto\n"},
		{"renomeie e descarte", "tenha clientes\n\ncada cliente tem\n    nome_completo\n\nrenomeie nome de clientes para nome_completo\ndescarte fax de clientes\n",
			"clientes\n    tem\n        nome_completo\n    renomeie nome para nome_completo\n    descarte fax\n"},
		{"singular", "tenha tokens de acesso\n\ncada token de acesso tem\n    nome\n",
			"tokens de acesso\n    singular token de acesso\n    tem\n        nome\n"},
		{"executam", "tenha trabalhadores, conversoes\n\ncada trabalhador tem\n    token secreto\n\ncada conversao tem\n    arquivo\n\ntrabalhadores executam conversoes\n",
			"trabalhadores\n    tem\n        token secreto\n    executam conversoes\n\nconversoes\n    tem\n        arquivo\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			flat := canonical(t, resolved(t, base+"\n"+c.flat))
			tree := canonical(t, resolved(t, base+"\n"+c.tree))
			if !reflect.DeepEqual(flat, tree) {
				fb, _ := json.MarshalIndent(flat, "", " ")
				tb, _ := json.MarshalIndent(tree, "", " ")
				t.Fatalf("formas diferentes:\n%s", diffHint(string(fb), string(tb)))
			}
		})
	}
}

func resolveErr(src string) error {
	toks, err := lexer.New(src).Tokenize()
	if err != nil {
		return err
	}
	prog, err := New(toks).Parse()
	if err != nil {
		return err
	}
	return ResolveIntent(prog)
}

// Erros da sintaxe hierárquica: cada um diz o quê, onde, por quê e como corrigir.
func TestHierarquiaErros(t *testing.T) {
	cases := []struct {
		name, src string
		want      []string
	}{
		{"recuo sem pai", "projetos\n    acesso\n        developer\n            ver\n      criar\n",
			[]string{"não tem um nível aberto", "Por quê", "Como corrigir", "4 = acesso", "8 = developer"}},
		{"seção desconhecida", "projetos\n    tem\n        nome\n    acessos\n        guest\n            ver\n",
			[]string{"\"acessos\" não é uma seção de projetos", "você quis dizer \"acesso\"", "Como corrigir"}},
		{"seção vazia", "projetos\n    tem\n", []string{"a seção tem está vazia"}},
		{"ator sem ações", "projetos\n    tem\n        nome\n    acesso\n        guest\n", []string{"\"guest\" não diz o que pode fazer", "ver"}},
		{"alvo de outro dado", "tenha labels\n\ncada label tem\n    nome\n\nprojetos\n    tem\n        nome\n    acesso\n        guest\n            ver labels\n",
			[]string{"está no bloco de projetos, mas fala de labels", "projetos › acesso › guest", "bloco de labels"}},
		{"linha desconhecida", "tenha issues\n\ncada issue tem\n    titulo\n\nissue podee fechar\n", []string{"não entendi a linha", "você quis dizer \"pode\""}},
		{"tab", "tenha issues\n\ncada issue tem\n\ttitulo\n", []string{"tab na indentação", "Como corrigir"}},
		{"estado inicial em conflito", "projetos\n    tem\n        nome\n    começa aberto\n\nprojeto começa fechado\n", []string{"já começa aberto", "(projetos › começa aberto)"}},
		{"linha sob uma ação de acesso", "projetos\n    tem\n        nome\n    acesso\n        guest\n            ver\n                criar\n",
			[]string{"\"criar\" está recuada abaixo de \"ver\"", "Como corrigir"}},
		{"linha sob um campo", "projetos\n    tem\n        nome\n            descrição\n",
			[]string{"\"descrição\" está recuada abaixo de \"nome\""}},
		{"linha sob uma ação de pode", "projetos\n    tem\n        nome\n    começa aberto\n    pode\n        fechar\n            reabrir\n",
			[]string{"\"reabrir\" está recuada abaixo de \"fechar\""}},
		{"linha sob o alvo de pertence a", "tenha grupos\n\ncada grupo tem\n    nome\n\nprojetos\n    tem\n        nome\n    pertence a\n        grupo\n            opcional\n",
			[]string{"\"opcional\" está recuada abaixo de \"grupo\""}},
		{"frase de intenção em inglês", "create system Shop\n", []string{"não entendi a linha", "só entende português", "crie sistema Shop"}},
		{"login com complemento", "tenha login com email e senha\n", []string{"mistura a declaração do login", "login usa email"}},
		{"por página no nível da página", "tenha clientes\n\ncada cliente tem\n    nome\n\npágina Clientes\n    mostre clientes\n    20 por página\n", []string{"está no nível da página", "abaixo de mostre"}},
		{"linha desconhecida sob mostre", "tenha clientes\n\ncada cliente tem\n    nome\n\npágina Clientes\n    mostre clientes\n        em ordem\n", []string{"não é algo que mostre aceite"}},
		{"rótulo sem verbo no topo", "tenha clientes\n\ncada cliente tem\n    nome\n    cidade\n    anotacao oculto\n\npágina Clientes\n    mostre clientes\n    topo\n        ações\n            \"Novo cliente\"\n    permita\n        criar\n", []string{"é só um rótulo", "criar \"Novo cliente\""}},
		{"ação não permitida pela página", "tenha clientes\n\ncada cliente tem\n    nome\n    cidade\n    anotacao oculto\n\npágina Clientes\n    mostre clientes\n    topo\n        ações\n            criar \"Novo\"\n", []string{"oferece a ação criar, mas não a permite"}},
		{"ação de registro no topo", "tenha clientes\n\ncada cliente tem\n    nome\n    cidade\n    anotacao oculto\n\npágina Clientes\n    mostre clientes\n    topo\n        ações\n            excluir\n    permita\n        excluir\n", []string{"só cabe a ação criar"}},
		{"coluna inexistente", "tenha clientes\n\ncada cliente tem\n    nome\n    cidade\n    anotacao oculto\n\npágina Clientes\n    mostre clientes\n    colunas\n        telefone\n", []string{"não tem esse campo"}},
		{"coluna secreta", "tenha clientes\n\ncada cliente tem\n    nome\n    cidade\n    anotacao oculto\n\npágina Clientes\n    mostre clientes\n    colunas\n        anotacao\n", []string{"privada ou secreta"}},
		{"seção de página vazia", "tenha clientes\n\ncada cliente tem\n    nome\n    cidade\n    anotacao oculto\n\npágina Clientes\n    mostre clientes\n    filtros\n", []string{"a seção filtros está vazia"}},
		{"rename para campo não declarado", "tenha clientes\n\ncada cliente tem\n    nome\n\nclientes\n    renomeie apelido para alcunha\n", []string{"alcunha não está declarado"}},
		{"rename com o antigo ainda declarado", "tenha clientes\n\ncada cliente tem\n    nome\n    nome_completo\n\nclientes\n    renomeie nome para nome_completo\n", []string{"nome continua declarado"}},
		{"dois renames para o mesmo campo", "tenha clientes\n\ncada cliente tem\n    nome_completo\n\nclientes\n    renomeie nome para nome_completo\n    renomeie apelido para nome_completo\n", []string{"não podem virar o mesmo campo"}},
		{"rename em cadeia", "tenha clientes\n\ncada cliente tem\n    terceiro\n\nrenomeie primeiro de clientes para segundo\nrenomeie segundo de clientes para terceiro\n", []string{"segundo não está declarado"}},
		{"rename sem para", "tenha clientes\n\ncada cliente tem\n    nome\n\nclientes\n    renomeie nome\n", []string{"não diz o que renomear"}},
		{"descarte de campo declarado", "tenha clientes\n\ncada cliente tem\n    nome\n\nclientes\n    descarte nome\n", []string{"nome continua declarado"}},
		{"resposta de efeito usada no hook", "tenha pedidos\n\npedido tem\n    cliente\n\nquando criar pedido\n    r = chamar(\"https://x.example\", \"POST\", pedido.cliente)\n", []string{"depois que a mudança é salva", "numa linha sozinha"}},
		{"efeito dentro de condição", "tenha pedidos\n\npedido tem\n    cliente\n\nquando criar pedido\n    se telegram_enviar(\"t\", \"oi\")\n        mostrar 1\n", []string{"telegram_enviar age fora do sistema"}},
		{"histórico com linhas abaixo", "tenha issues\n\nissues\n    tem\n        titulo\n    guarda histórico\n        titulo\n", []string{"guarda histórico fica sozinho"}},
		{"histórico de dado desconhecido", "tenha issues\n\ncada issue tem\n    titulo\n\nnoticia guarda histórico\n", []string{"noticia"}},
		{"tradução repetida com outro valor", "tenha clientes\n\ncada cliente tem\n    nome\n\nvocabulário da integração\n    nome é \"name\"\n    nome é \"title\"\n", []string{"nome já é traduzido como \"name\""}},
		{"indicador sem total de", "tenha issues\n\ncada issue tem\n    titulo\n\npágina Painel\n    indicadores\n        issues\n", []string{"não é um indicador", "total de issues"}},
		{"indicador com estado inexistente", "tenha issues\n\ncada issue tem\n    titulo\n\nissue começa aberta\n\npágina Painel\n    indicadores\n        total de issues voadoras\n", []string{"\"voadoras\" não é um estado de issues", "aberta"}},
		{"página sem nada", "tenha issues\n\ncada issue tem\n    titulo\n\npágina Vazia\n    permita\n        criar\n", []string{"não mostra nada"}},
		{"nome de dado impossível", "tenha clientes e -x\n", []string{"não pode ser o nome de um dado"}},
		{"avisos sem pendências", "tenha issues\n\ncada issue tem\n    titulo\n\ntenha avisos por e-mail\n", []string{"nenhum dado gera pendências"}},
		{"variáveis de outro dono", "projetos\n    tem\n        caminho único\n        pipelines\n    tem repositório\n    executa pipelines a cada envio de código conforme \"ci.yml\"\n\npipelines\n    tem\n        jobs\n\njobs\n    tem\n        nome\n\nvariaveis\n    tem\n        chave\n        valor texto oculto\n    pertence a projeto\n\npipelines usam as variaveis do grupo\n", []string{"são executados por projeto"}},
		{"valor pelo nome é dinheiro", "projetos\n    tem\n        caminho único\n        pipelines\n    tem repositório\n    executa pipelines a cada envio de código conforme \"ci.yml\"\n\npipelines\n    tem\n        jobs\n\njobs\n    tem\n        nome\n\nvariaveis\n    tem\n        chave\n        valor oculto\n    pertence a projeto\n\npipelines usam as variaveis do projeto\n", []string{"valor texto oculto"}},
		{"mencionados sem texto", "tenha chamados\n\ncada chamado tem\n    titulo\n\nchamado gera pendência para mencionados\n", []string{"não tem texto onde alguém seja mencionado"}},
		{"mencionados sem nome de usuário", "tenha chamados\n\ncada chamado tem\n    descricao\n\nchamado gera pendência para mencionados\n", []string{"nome de usuário único"}},
		{"por estado sem estados", "tenha clientes\n\ncada cliente tem\n    nome\n\npágina Clientes\n    mostre clientes por estado\n", []string{"clientes não tem estados"}},
		{"integração em conflito", "projetos\n    tem\n        nome\n    integração\n        nome \"projects\"\n\ndisponibilize projetos para integração como \"repos\"\n", []string{"já é \"projects\""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := resolveErr(base + "\n" + c.src)
			if err == nil {
				t.Fatalf("esperado erro para:\n%s", c.src)
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("mensagem sem %q:\n%v", w, err)
				}
			}
		})
	}
}

// Fusão: o mesmo dado em dois blocos (e em frase plana) soma os fatos; o
// mesmo fato repetido não muda nada.
// Repeating the same field in another block changes nothing; two different
// definitions of the same field are a conflict that shows both origins.
func TestHierarquiaFusaoDeCampos(t *testing.T) {
	once := canonical(t, resolved(t, base+"\nprojetos\n    tem\n        nome obrigatório\n"))
	twice := canonical(t, resolved(t, base+"\nprojetos\n    tem\n        nome obrigatório\n\nprojetos\n    tem\n        nome obrigatório\n"))
	if !reflect.DeepEqual(once, twice) {
		t.Fatal("repetir o mesmo campo mudou o programa")
	}
	err := resolveErr(base + "\nprojetos\n    tem\n        nome obrigatório\n\nprojetos\n    tem\n        nome até 10\n")
	if err == nil {
		t.Fatal("duas definições diferentes do mesmo campo foram aceitas")
	}
	for _, want := range []string{"nome", "já foi definido", "linha"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("mensagem sem %q: %v", want, err)
		}
	}
	if strings.Count(err.Error(), "linha") < 2 && strings.Count(err.Error(), ":") < 2 {
		t.Fatalf("a mensagem não mostra as duas origens: %v", err)
	}
}

func TestHierarquiaFusao(t *testing.T) {
	one := canonical(t, resolved(t, base+"\nprojetos\n    tem\n        nome\n    começa aberto\n    pode\n        fechar\n    acesso\n        guest\n            ver\n"))
	split := canonical(t, resolved(t, base+"\nprojetos\n    tem\n        nome\n    começa aberto\n\nprojetos\n    pode\n        fechar\n\nprojeto começa aberto\nguest pode ver projetos\n"))
	if !reflect.DeepEqual(one, split) {
		a, _ := json.MarshalIndent(one, "", " ")
		b, _ := json.MarshalIndent(split, "", " ")
		t.Fatalf("fusão diferente:\n%s", diffHint(string(a), string(b)))
	}
}

// Presence is about the people who log in (GEP 0021).
func TestPresencaSemLogin(t *testing.T) {
	err := resolveErr("crie sistema x\n\ntenha clientes\n\ncada cliente tem\n    nome\n\ntenha presença\n")
	if err == nil || !strings.Contains(err.Error(), "declare também tenha login") {
		t.Fatalf("presença sem login: %v", err)
	}
}
