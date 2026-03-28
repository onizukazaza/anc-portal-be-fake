package sync

import "fmt"

// Registry เก็บ syncer ทั้งหมด — เพิ่มตารางใหม่แค่เรียก Register().
type Registry struct {
	syncers map[string]Syncer
}

// NewRegistry สร้าง empty registry.
func NewRegistry() *Registry {
	return &Registry{syncers: make(map[string]Syncer)}
}

// Register ลงทะเบียน syncer ด้วย name ของมัน.
func (r *Registry) Register(s Syncer) {
	r.syncers[s.Name()] = s
}

// Get ดึง syncer ตามชื่อ.
func (r *Registry) Get(name string) (Syncer, error) {
	s, ok := r.syncers[name]
	if !ok {
		return nil, fmt.Errorf("syncer not found: %s (available: %v)", name, r.Names())
	}
	return s, nil
}

// Names คืนชื่อ syncer ทั้งหมดที่ลงทะเบียนไว้.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.syncers))
	for k := range r.syncers {
		names = append(names, k)
	}
	return names
}

// All คืน syncer ทั้งหมด.
func (r *Registry) All() []Syncer {
	all := make([]Syncer, 0, len(r.syncers))
	for _, s := range r.syncers {
		all = append(all, s)
	}
	return all
}
