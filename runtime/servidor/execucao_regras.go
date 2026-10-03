package servidor

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Flow of a run, in the native format (see readRunFile): which steps exist
// on a branch (somente_em, exceto_em, regras), what a step waits for
// (precisa: a graph instead of whole stages), whose files it receives
// (recebe_artefatos_de) and how long its own files are kept
// (artefatos_expiram_em). Nothing here knows any external CI format; an
// adapter translates one into these keys.

const whenNever = "nunca" // only in regras: the step is left out

// mainBranch is the word for the owner's main branch in branch lists. A
// branch name never has spaces, so it cannot collide with a real branch.
const mainBranch = "branch padrão"

// need is one entry of precisa.
type need struct {
	Name     string
	Optional bool // when the needed step is left out on this branch, do not wait for it
}

// stepRule is one entry of regras: the first that matches the branch decides.
type stepRule struct {
	In, Except   []string
	InSet        bool
	When         string
	AllowFailure *bool
}

// branchMatches: pattern is a branch name, `branch padrão`, or a name with
// `*` standing for any text (`release/*` matches release/1.0 and release/a/b).
func branchMatches(pattern, branch, main string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == mainBranch {
		return branch == main
	}
	if !strings.Contains(pattern, "*") {
		return pattern == branch
	}
	re := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, ".*") + "$"
	ok, _ := regexp.MatchString(re, branch)
	return ok
}

func anyBranch(patterns []string, branch, main string) bool {
	for _, p := range patterns {
		if branchMatches(p, branch, main) {
			return true
		}
	}
	return false
}

// branchList reads somente_em / exceto_em / em: a list of branch patterns.
// present says whether the key was written (an empty list means "none").
func branchList(job map[string]any, key, step string) (list []string, present bool, err error) {
	v, ok := job[key]
	if !ok || v == nil {
		return nil, false, nil
	}
	switch x := v.(type) {
	case string:
		list = []string{x}
	case []any:
		for _, it := range x {
			s, ok := it.(string)
			if !ok || strings.TrimSpace(s) == "" {
				return nil, true, fmt.Errorf("configuração inválida: a etapa %s tem em %s um item que não é nome de branch (%v); escreva nomes como main ou release/*", step, key, it)
			}
			list = append(list, strings.TrimSpace(s))
		}
	default:
		return nil, true, fmt.Errorf("configuração inválida: a etapa %s tem %s que não é uma lista de branches; escreva por exemplo %s: [main]", step, key, key)
	}
	return list, true, nil
}

// yes reads a yes/no value: YAML booleans or the words sim/não.
func yes(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "sim", "verdadeiro":
			return true, true
		case "não", "nao", "falso":
			return false, true
		}
	}
	return false, false
}

// parseFlow reads the flow keys of one step into sp.
func parseFlow(sp *stepSpec, job map[string]any) error {
	name := sp.Name
	var err error
	if sp.Only, sp.OnlySet, err = branchList(job, "somente_em", name); err != nil {
		return err
	}
	if sp.Except, _, err = branchList(job, "exceto_em", name); err != nil {
		return err
	}
	if v, ok := job["regras"]; ok && v != nil {
		list, ok := v.([]any)
		if !ok {
			return fmt.Errorf("configuração inválida: a etapa %s tem regras que não são uma lista; escreva cada regra como um item: regras: [{em: [main], quando: manual}]", name)
		}
		for i, it := range list {
			m, ok := it.(map[string]any)
			if !ok {
				return fmt.Errorf("configuração inválida: a regra %d da etapa %s deve ser um mapa com em, exceto_em, quando ou pode_falhar", i+1, name)
			}
			for k := range m {
				switch k {
				case "em", "exceto_em", "quando", "pode_falhar":
				default:
					return fmt.Errorf("configuração inválida: a regra %d da etapa %s tem a chave %q; use em, exceto_em, quando ou pode_falhar", i+1, name, k)
				}
			}
			r := stepRule{When: sp.When}
			if r.In, r.InSet, err = branchList(m, "em", name); err != nil {
				return err
			}
			if r.Except, _, err = branchList(m, "exceto_em", name); err != nil {
				return err
			}
			if w, ok := m["quando"]; ok {
				s, _ := w.(string)
				switch s {
				case whenAuto, whenManual, whenAlways, whenNever:
					r.When = s
				default:
					return fmt.Errorf("configuração inválida: a regra %d da etapa %s tem quando %q (use automatico, manual, sempre ou nunca)", i+1, name, s)
				}
			}
			if v, ok := m["pode_falhar"]; ok {
				b, valid := yes(v)
				if !valid {
					return fmt.Errorf("configuração inválida: a regra %d da etapa %s tem pode_falhar %v (use sim ou não)", i+1, name, v)
				}
				r.AllowFailure = &b
			}
			sp.Rules = append(sp.Rules, r)
		}
	}
	if v, ok := job["precisa"]; ok && v != nil {
		sp.NeedsSet = true
		var items []any
		switch x := v.(type) {
		case string:
			items = []any{x}
		case []any:
			items = x
		default:
			return fmt.Errorf("configuração inválida: a etapa %s tem precisa que não é uma lista de etapas; escreva por exemplo precisa: [compilar]", name)
		}
		for _, it := range items {
			switch x := it.(type) {
			case string:
				sp.Needs = append(sp.Needs, need{Name: x})
			case map[string]any:
				n, _ := x["etapa"].(string)
				if n == "" {
					return fmt.Errorf("configuração inválida: a etapa %s tem em precisa um item sem etapa; escreva {etapa: compilar, opcional: sim}", name)
				}
				opt, _ := yes(x["opcional"])
				sp.Needs = append(sp.Needs, need{Name: n, Optional: opt})
			default:
				return fmt.Errorf("configuração inválida: a etapa %s tem em precisa o item %v; escreva o nome de uma etapa", name, it)
			}
		}
	}
	if v, ok := job["recebe_artefatos_de"]; ok && v != nil {
		sp.FromSet = true
		sp.From = lines(v)
		if _, isList := v.([]any); !isList {
			if _, isText := v.(string); !isText {
				return fmt.Errorf("configuração inválida: a etapa %s tem recebe_artefatos_de que não é uma lista de etapas", name)
			}
		}
	}
	if v, ok := job["artefatos_expiram_em"]; ok && v != nil {
		d, err := parseKeep(v)
		if err != nil {
			return fmt.Errorf("configuração inválida: a etapa %s tem artefatos_expiram_em %v: %v", name, v, err)
		}
		sp.Expire = d
	}
	return nil
}

var keepUnits = map[string]time.Duration{
	"segundo": time.Second, "segundos": time.Second,
	"minuto": time.Minute, "minutos": time.Minute,
	"hora": time.Hour, "horas": time.Hour,
	"dia": 24 * time.Hour, "dias": 24 * time.Hour,
	"semana": 7 * 24 * time.Hour, "semanas": 7 * 24 * time.Hour,
	"mês": 30 * 24 * time.Hour, "mes": 30 * 24 * time.Hour, "meses": 30 * 24 * time.Hour,
	"ano": 365 * 24 * time.Hour, "anos": 365 * 24 * time.Hour,
}

// parseKeep reads how long files are kept: "7 dias", "1 hora 30 minutos" or
// "nunca" (0: kept until the step is removed). A month is 30 days, a year 365.
func parseKeep(v any) (time.Duration, error) {
	s, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("diga o número e a unidade, por exemplo \"7 dias\"")
	}
	words := strings.Fields(strings.ToLower(s))
	if len(words) == 1 && words[0] == "nunca" {
		return 0, nil
	}
	if len(words) == 0 || len(words)%2 != 0 {
		return 0, fmt.Errorf("diga o número e a unidade, por exemplo \"7 dias\" ou \"1 hora 30 minutos\" (ou \"nunca\")")
	}
	var total time.Duration
	for i := 0; i < len(words); i += 2 {
		n, err := strconv.Atoi(words[i])
		if err != nil || n < 0 {
			return 0, fmt.Errorf("%q não é um número inteiro; escreva por exemplo \"7 dias\"", words[i])
		}
		unit, ok := keepUnits[words[i+1]]
		if !ok {
			return 0, fmt.Errorf("unidade %q desconhecida; use segundos, minutos, horas, dias, semanas, meses ou anos", words[i+1])
		}
		total += time.Duration(n) * unit
	}
	if total <= 0 {
		return 0, fmt.Errorf("o tempo precisa ser maior que zero (ou \"nunca\")")
	}
	return total, nil
}

// checkGraph validates precisa and recebe_artefatos_de: every name is a
// step of the file, and no step waits for itself, directly or not.
func checkGraph(specs []stepSpec) error {
	index := map[string]int{}
	for i, s := range specs {
		index[s.Name] = i
	}
	for _, s := range specs {
		for _, n := range s.Needs {
			if n.Name == s.Name {
				return fmt.Errorf("configuração inválida: a etapa %s não pode precisar de si mesma", s.Name)
			}
			if _, ok := index[n.Name]; !ok {
				return fmt.Errorf("configuração inválida: a etapa %s precisa de %q, que não é uma etapa deste arquivo", s.Name, n.Name)
			}
		}
		for _, n := range s.From {
			if _, ok := index[n]; !ok || n == s.Name {
				return fmt.Errorf("configuração inválida: a etapa %s recebe artefatos de %q, que não é outra etapa deste arquivo", s.Name, n)
			}
		}
	}
	// depth-first search for a cycle, reporting the path
	state := make([]int, len(specs)) // 0 unseen, 1 on the path, 2 done
	var path []string
	var visit func(i int) error
	visit = func(i int) error {
		state[i] = 1
		path = append(path, specs[i].Name)
		for _, n := range specs[i].Needs {
			j := index[n.Name]
			switch state[j] {
			case 1:
				return fmt.Errorf("configuração inválida: as etapas esperam umas pelas outras em círculo (%s → %s); nenhuma poderia começar", strings.Join(path, " → "), n.Name)
			case 0:
				if err := visit(j); err != nil {
					return err
				}
			}
		}
		path = path[:len(path)-1]
		state[i] = 2
		return nil
	}
	for i := range specs {
		if state[i] == 0 {
			if err := visit(i); err != nil {
				return err
			}
		}
	}
	return nil
}

// errNoSteps: no step of the file is meant for this branch, so no run is created.
type errNoSteps struct{ branch string }

func (e errNoSteps) Error() string {
	return fmt.Sprintf("nenhuma etapa vale para a branch %s (veja somente_em, exceto_em e regras)", e.branch)
}

// forBranch keeps the steps meant for branch, with the when and allow
// failure their rules decide, and drops optional needs left out.
func forBranch(specs []stepSpec, branch, main string) ([]stepSpec, error) {
	var out []stepSpec
	kept := map[string]bool{}
	for _, s := range specs {
		if s.OnlySet && !anyBranch(s.Only, branch, main) {
			continue
		}
		if anyBranch(s.Except, branch, main) {
			continue
		}
		if len(s.Rules) > 0 {
			matched := false
			for _, r := range s.Rules {
				if r.InSet && !anyBranch(r.In, branch, main) || anyBranch(r.Except, branch, main) {
					continue
				}
				matched = true
				s.When = r.When
				if r.AllowFailure != nil {
					s.AllowFailure = *r.AllowFailure
				}
				break
			}
			if !matched || s.When == whenNever {
				continue
			}
		}
		out = append(out, s)
		kept[s.Name] = true
	}
	if len(out) == 0 {
		return nil, errNoSteps{branch}
	}
	for i := range out {
		var needs []need
		for _, n := range out[i].Needs {
			if kept[n.Name] {
				needs = append(needs, n)
			} else if !n.Optional {
				return nil, fmt.Errorf("configuração inválida: a etapa %s precisa de %s, que não vale para a branch %s (marque {etapa: %s, opcional: sim} ou ajuste as regras)", out[i].Name, n.Name, branch, n.Name)
			}
		}
		out[i].Needs = needs
		var from []string
		for _, n := range out[i].From {
			if kept[n] {
				from = append(from, n)
			}
		}
		out[i].From = from
	}
	return out, nil
}
