package eval

type Value interface{}

type Scope struct {
	parent *Scope
	vars   map[string]Value
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent: parent,
		vars:   make(map[string]Value),
	}
}

func NewGlobalScope() *Scope {
	s := NewScope(nil)
	s.vars["?"] = int64(0)
	s.vars["stderr"] = ""
	s.vars["stdout"] = ""
	s.vars["report"] = ""
	return s
}

func (s *Scope) Get(name string) (Value, bool) {
	v, ok := s.vars[name]
	if !ok && s.parent != nil {
		return s.parent.Get(name)
	}
	return v, ok
}

func (s *Scope) Set(name string, val Value) {
	if s.parent != nil {
		if _, exists := s.vars[name]; !exists {
			if _, parentExists := s.parent.vars[name]; parentExists {
				s.parent.Set(name, val)
				return
			}
		}
	}
	s.vars[name] = val
}

func (s *Scope) SetLocal(name string, val Value) {
	s.vars[name] = val
}

func (s *Scope) SetReadonly(name string) {
	s.vars["__readonly__"+name] = struct{}{}
}

func (s *Scope) IsReadonly(name string) bool {
	if s.parent != nil {
		_, inParent := s.parent.vars["__readonly__"+name]
		if inParent {
			return true
		}
	}
	_, ok := s.vars["__readonly__"+name]
	return ok
}

func (s *Scope) HasLocal(name string) bool {
	_, ok := s.vars[name]
	return ok
}
