package prober

type Probe interface {
	Healthy()
	NotHealthy(err error)
	Ready()
	NotReady(err error)
}
