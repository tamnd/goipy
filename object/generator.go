package object

// Generator is a suspended Python generator. Frame is stored as `any` to
// avoid an import cycle with vm; the vm package is the only thing that ever
// unwraps it, and it always holds a *vm.Frame.
type Generator struct {
	Name    string
	Frame   any
	Started bool
	Done    bool
	// Running is set true while the dispatcher is actively stepping the
	// generator's frame. Surfaces through gi_running / cr_running / ag_running.
	Running bool
}
