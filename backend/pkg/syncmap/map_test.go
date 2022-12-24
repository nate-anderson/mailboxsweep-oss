package syncmap

import "testing"

type testStruct struct {
	A    string
	B    string
	List []string
}

func TestSyncMap(t *testing.T) {
	m := New[string, *testStruct]()

	t.Run("set value", func(t *testing.T) {
		m.Set("hello", &testStruct{
			A:    "buddy",
			B:    "pal",
			List: []string{"friend", "amigo"},
		})

		if val, ok := m.m["hello"]; !ok {
			t.Errorf("'hello' wasn't stored")
		} else {
			if val.A != "buddy" || val.B != "pal" || val.List[0] != "friend" || val.List[1] != "amigo" {
				t.Errorf("mismatched struct: got %+v", val)
			}
		}
	})

	t.Run("get value", func(t *testing.T) {
		val, ok := m.Get("hello")
		if !ok {
			t.Errorf("no value returned")
		}
		if val.A != "buddy" || val.B != "pal" || val.List[0] != "friend" || val.List[1] != "amigo" {
			t.Errorf("mismatched struct: got %+v", val)
		}
	})

	t.Run("update value", func(t *testing.T) {
		n := &testStruct{}
		m.Set("hello", n)

		r, ok := m.Get("hello")
		if !ok {
			t.Errorf("value not returned")
		}
		if r != n {
			t.Errorf("value %+v != expected %v", n, r)
		}
	})
}
