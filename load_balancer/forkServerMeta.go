package load_balancer

type ForkServerMeta struct {
	Imports map[string]bool
	Hits    float64
}

func (fsm *ForkServerMeta) Hit(hitForkServerMeta bool) {
	fsm.Hits *= 0.9
	if hitForkServerMeta {
		fsm.Hits += 1
	}

	return
}

