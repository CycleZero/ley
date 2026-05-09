package service

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toUintPtr(v uint64) *uint {
	if v == 0 {
		return nil
	}
	u := uint(v)
	return &u
}

func derefUint(p *uint) uint64 {
	if p == nil {
		return 0
	}
	return uint64(*p)
}
