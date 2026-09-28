package ast

import (
	"reflect"
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/diagnostics"
)

// filled returns a non-zero value of type t (one element for slices and
// maps, a new value for pointers).
func filled(t reflect.Type) reflect.Value {
	switch t.Kind() {
	case reflect.Slice:
		return reflect.MakeSlice(t, 1, 1)
	case reflect.Map:
		m := reflect.MakeMapWithSize(t, 1)
		m.SetMapIndex(reflect.Zero(t.Key()), reflect.Zero(t.Elem()))
		return m
	case reflect.Ptr:
		return reflect.New(t.Elem())
	case reflect.String:
		return reflect.ValueOf("x").Convert(t)
	case reflect.Bool:
		return reflect.ValueOf(true)
	case reflect.Int:
		return reflect.ValueOf(1).Convert(t)
	case reflect.Interface:
		return reflect.ValueOf("x")
	}
	return reflect.Zero(t)
}

// Every declaration survives when the program is split in files: for each
// field of Intent (and of the login declaration), a value declared only in a
// second file is present after merging. A field added later that the merge
// forgets fails here, not in production (the password-recovery flag and the
// pending rules were once lost this way).
func TestMergeIntentKeepsEveryField(t *testing.T) {
	it := reflect.TypeOf(Intent{})
	for i := 0; i < it.NumField(); i++ {
		f := it.Field(i)
		if !f.IsExported() {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			a, b := &Intent{}, &Intent{}
			reflect.ValueOf(b).Elem().Field(i).Set(filled(f.Type))
			m := MergeIntent(a, b)
			if reflect.ValueOf(m).Elem().Field(i).IsZero() {
				t.Fatalf("Intent.%s declarado num segundo arquivo se perdeu na fusão", f.Name)
			}
		})
	}
	lt := reflect.TypeOf(LoginDecl{})
	for i := 0; i < lt.NumField(); i++ {
		f := lt.Field(i)
		if !f.IsExported() || f.Name == "Pos" {
			continue
		}
		t.Run("Login."+f.Name, func(t *testing.T) {
			a, b := &Intent{Login: &LoginDecl{}}, &Intent{Login: &LoginDecl{}}
			reflect.ValueOf(b.Login).Elem().Field(i).Set(filled(f.Type))
			m := MergeIntent(a, b)
			if reflect.ValueOf(m.Login).Elem().Field(i).IsZero() {
				t.Fatalf("Login.%s declarado num segundo arquivo se perdeu na fusão", f.Name)
			}
		})
	}
}

// Two files translating the same name differently: the merge records the
// conflict with both origins instead of letting one win in silence.
func TestMergeVocabularioEmConflito(t *testing.T) {
	a := &Intent{Vocabulary: map[string]string{"acao": "action_name"}, VocabularyPos: map[string]diagnostics.Position{"acao": {File: "a.ge", Line: 3}}}
	b := &Intent{Vocabulary: map[string]string{"acao": "event_name"}, VocabularyPos: map[string]diagnostics.Position{"acao": {File: "b.ge", Line: 9}}}
	m := MergeIntent(a, b)
	if len(m.Conflicts) != 1 || !strings.Contains(m.Conflicts[0], "a.ge:3") || !strings.Contains(m.Conflicts[0], "b.ge:9") || m.Vocabulary["acao"] != "action_name" {
		t.Fatalf("conflito de vocabulário: %v %v", m.Conflicts, m.Vocabulary)
	}
	same := MergeIntent(&Intent{Vocabulary: map[string]string{"acao": "x"}}, &Intent{Vocabulary: map[string]string{"acao": "x"}})
	if len(same.Conflicts) != 0 {
		t.Fatalf("a mesma tradução duas vezes não é conflito: %v", same.Conflicts)
	}
}
