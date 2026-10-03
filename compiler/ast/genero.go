package ast

import "strings"

// NovoRotulo is the label of the action that creates a record, agreeing
// in gender with the name of the data: "Novo pedido", "Nova tarefa".
func (e *Entity) NovoRotulo() string {
	nome := strings.ToLower(e.Label)
	if Feminino(nome, e.Initial) {
		return "Nova " + nome
	}
	return "Novo " + nome
}

// Concorda picks the masculine or the feminine word to agree with the name
// of the data: Concorda("criado", "criada") is "criada" for a tarefa.
func (e *Entity) Concorda(masculino, feminino string) string {
	if Feminino(e.Label, e.Initial) {
		return feminino
	}
	return masculino
}

// Feminino decides, deterministically, whether a name is feminine in
// Portuguese. The words the person wrote come first: an initial state that
// agrees with the name (tarefa começa aberta, issue começa aberta) decides.
// Without it, the ending of the first word decides (-a, -ção, -são, -dade,
// -gem, -tude, -ie are feminine), with the common masculine words ending in
// -a as exceptions (sistema, dia, problema). Everything else is masculine.
func Feminino(nome, estadoInicial string) bool {
	if g, ok := generoDoParticipio(dobrar(estadoInicial)); ok {
		return g
	}
	palavra := dobrar(nome)
	if i := strings.IndexByte(palavra, ' '); i >= 0 {
		palavra = palavra[:i]
	}
	if masculinosEmA[palavra] {
		return false
	}
	for _, fim := range []string{"a", "cao", "sao", "dade", "gem", "tude", "ie"} {
		if strings.HasSuffix(palavra, fim) {
			return true
		}
	}
	return false
}

// generoDoParticipio reads the gender of a participle or adjective state
// (aberta, fechado, concluída, ativa, impresso); a noun state (rascunho,
// fila) says nothing.
func generoDoParticipio(estado string) (feminino, ok bool) {
	if len(estado) < 3 {
		return false, false
	}
	fim, antes := estado[len(estado)-1], estado[len(estado)-2]
	if (fim != 'a' && fim != 'o') || !strings.ContainsRune("dtsv", rune(antes)) {
		return false, false
	}
	return fim == 'a', true
}

var masculinosEmA = map[string]bool{
	"dia": true, "mapa": true, "sistema": true, "problema": true, "tema": true,
	"programa": true, "clima": true, "idioma": true, "planeta": true, "cometa": true,
	"esquema": true, "diagrama": true, "telefonema": true, "poema": true, "pijama": true,
	"sofa": true, "lema": true, "dilema": true, "teorema": true,
	"sintoma": true, "trauma": true, "drama": true, "prisma": true, "enigma": true,
	"dogma": true, "aroma": true, "cinema": true, "fantasma": true, "panorama": true,
	"alfa": true, "delta": true, "gama": true,
}

// dobrar lowers a word and removes its Portuguese accents.
func dobrar(s string) string {
	return dobrador.Replace(strings.ToLower(strings.TrimSpace(s)))
}

// dobrador is built once (a Replacer is safe for concurrent use).
var dobrador = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "ê", "e", "í", "i",
	"ó", "o", "ô", "o", "õ", "o", "ú", "u", "ü", "u", "ç", "c",
)
