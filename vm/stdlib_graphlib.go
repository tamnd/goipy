package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildGraphlib() *object.Module {
	m := &object.Module{Name: "graphlib", Dict: object.NewDict()}

	// CycleError is a subclass of ValueError.
	cycleErrClass := &object.Class{
		Name:  "CycleError",
		Bases: []*object.Class{i.valueErr},
		Dict:  object.NewDict(),
	}
	m.Dict.SetStr("CycleError", cycleErrClass)

	// TopologicalSorter state constants.
	const (
		tsStateNew      = 0 // before prepare()
		tsStatePrepared = 1 // after prepare(), before any get_ready()/done()
		tsStateActive   = 2 // after first get_ready() or done()
		tsStateDone     = 3 // all nodes done
	)

	// TopologicalSorter class.
	tsCls := &object.Class{Name: "TopologicalSorter", Dict: object.NewDict()}

	// Internal per-instance state is stored in the instance dict.
	// Keys: _graph_ (map node->predecessors), _state_, _ready_, _out_degree_, _n2i_ (name->node object)

	// Helper: get node's string key for internal maps.
	// We store nodes by their Repr (stable and hashable for our purposes).
	// We use the object itself as key in a *object.Dict for O(1) lookup.

	// __init__(self, graph=None)
	tsCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "TopologicalSorter.__init__ requires self")
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "self must be an instance")
		}

		// _graph_: Dict mapping node -> Set of predecessor nodes
		graph := object.NewDict()
		// _in_degree_: Dict mapping node -> int count of unsatisfied predecessors
		inDeg := object.NewDict()
		// _dependents_: Dict mapping node -> list of nodes that depend on it
		dependents := object.NewDict()
		// _ready_: list of nodes with zero in-degree (after prepare)
		ready := &object.List{}
		// _out_ready_: list of nodes returned by get_ready but not yet done
		outReady := &object.List{}
		// _state_: int
		state := object.NewInt(int64(tsStateNew))

		self.Dict.SetStr("_graph_", graph)
		self.Dict.SetStr("_in_degree_", inDeg)
		self.Dict.SetStr("_dependents_", dependents)
		self.Dict.SetStr("_ready_", ready)
		self.Dict.SetStr("_out_ready_", outReady)
		self.Dict.SetStr("_state_", state)
		self.Dict.SetStr("_n_done_", object.NewInt(0))
		self.Dict.SetStr("_n_total_", object.NewInt(0))

		// If graph argument provided, call add() for each entry.
		if len(a) >= 2 && a[1] != object.None {
			graphArg, ok := a[1].(*object.Dict)
			if !ok {
				return nil, object.Errorf(i.typeErr, "graph argument must be a dict or None")
			}
			keys, vals := graphArg.Items()
			for idx, k := range keys {
				preds := vals[idx]
				// Build args: self, node, *predecessors
				addArgs := []object.Object{self, k}
				switch p := preds.(type) {
				case *object.List:
					addArgs = append(addArgs, p.V...)
				case *object.Tuple:
					addArgs = append(addArgs, p.V...)
				case *object.Set:
					for _, item := range setItems(p) {
						addArgs = append(addArgs, item)
					}
				default:
					return nil, object.Errorf(i.typeErr, "graph values must be iterables")
				}
				if addFn, ok := tsCls.Dict.GetStr("add"); ok {
					if _, err := i.callObject(addFn, addArgs, nil); err != nil {
						return nil, err
					}
				}
			}
		}
		return object.None, nil
	}})

	// add(self, node, *predecessors)
	tsCls.Dict.SetStr("add", &object.BuiltinFunc{Name: "add", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "add() requires at least node argument")
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "self must be an instance")
		}
		// Check state.
		stateObj, _ := self.Dict.GetStr("_state_")
		stateInt, _ := toInt64(stateObj)
		if stateInt != int64(tsStateNew) {
			return nil, object.Errorf(i.valueErr, "add() cannot be called after prepare()")
		}

		node := a[1]
		predecessors := a[2:]

		graph, _ := self.Dict.GetStr("_graph_")
		graphDict := graph.(*object.Dict)

		// Ensure node exists in graph.
		if _, found, _ := graphDict.Get(node); !found {
			graphDict.Set(node, &object.List{})
		}

		// Add each predecessor.
		for _, pred := range predecessors {
			// Ensure pred exists in graph.
			if _, found, _ := graphDict.Get(pred); !found {
				graphDict.Set(pred, &object.List{})
			}
			// Add pred to node's predecessor list (avoid duplicates).
			predListObj, _, _ := graphDict.Get(node)
			predList := predListObj.(*object.List)
			alreadyHave := false
			for _, existing := range predList.V {
				if eq, _ := object.Eq(existing, pred); eq {
					alreadyHave = true
					break
				}
			}
			if !alreadyHave {
				predList.V = append(predList.V, pred)
			}
		}
		return object.None, nil
	}})

	// prepare()
	tsCls.Dict.SetStr("prepare", &object.BuiltinFunc{Name: "prepare", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "prepare() requires self")
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "self must be an instance")
		}
		stateObj, _ := self.Dict.GetStr("_state_")
		stateInt, _ := toInt64(stateObj)
		if stateInt == int64(tsStateActive) {
			return nil, object.Errorf(i.valueErr, "cannot prepare() after iteration has started")
		}

		graph, _ := self.Dict.GetStr("_graph_")
		graphDict := graph.(*object.Dict)
		nodes, predLists := graphDict.Items()
		n := len(nodes)

		// Build in-degree and dependents maps.
		inDeg := object.NewDict()
		dependents := object.NewDict()
		for _, nd := range nodes {
			inDeg.Set(nd, object.NewInt(0))
			dependents.Set(nd, &object.List{})
		}
		for idx, nd := range nodes {
			preds := predLists[idx].(*object.List)
			for _, pred := range preds.V {
				// nd depends on pred → increment nd's in-degree
				curObj, _, _ := inDeg.Get(nd)
				cur, _ := toInt64(curObj)
				inDeg.Set(nd, object.NewInt(cur+1))
				// pred's dependents includes nd
				depListObj, _, _ := dependents.Get(pred)
				depList := depListObj.(*object.List)
				depList.V = append(depList.V, nd)
			}
		}

		// Cycle detection via DFS.
		// 0=unvisited, 1=in-stack, 2=done
		visited := map[string]int{} // using Repr key
		nodeReprMap := map[string]object.Object{}
		for _, nd := range nodes {
			r := object.Repr(nd)
			nodeReprMap[r] = nd
			visited[r] = 0
		}

		var cycleFound []object.Object
		var dfs func(nd object.Object, stack []object.Object) bool
		dfs = func(nd object.Object, stack []object.Object) bool {
			r := object.Repr(nd)
			visited[r] = 1
			stack = append(stack, nd)
			predListObj, _, _ := graphDict.Get(nd)
			if predListObj == nil {
				visited[r] = 2
				return false
			}
			preds := predListObj.(*object.List)
			for _, pred := range preds.V {
				pr := object.Repr(pred)
				if visited[pr] == 1 {
					// Found cycle: extract cycle portion of stack.
					cycleStart := -1
					for ci, sn := range stack {
						if eq, _ := object.Eq(sn, pred); eq {
							cycleStart = ci
							break
						}
					}
					if cycleStart < 0 {
						cycleStart = 0
					}
					cycle := make([]object.Object, 0, len(stack)-cycleStart+1)
					cycle = append(cycle, stack[cycleStart:]...)
					cycle = append(cycle, pred) // close the cycle
					cycleFound = cycle
					return true
				}
				if visited[pr] == 0 {
					if dfs(pred, stack) {
						return true
					}
				}
			}
			visited[r] = 2
			return false
		}

		for _, nd := range nodes {
			r := object.Repr(nd)
			if visited[r] == 0 {
				if dfs(nd, nil) {
					break
				}
			}
		}

		if cycleFound != nil {
			cycleList := &object.List{V: cycleFound}
			exc := object.Errorf(cycleErrClass, "nodes are in a cycle")
			exc.Args = &object.Tuple{V: []object.Object{
				&object.Str{V: "nodes are in a cycle"},
				cycleList,
			}}
			return nil, exc
		}

		// Populate ready list: nodes with in-degree 0.
		ready := &object.List{}
		for _, nd := range nodes {
			curObj, _, _ := inDeg.Get(nd)
			cur, _ := toInt64(curObj)
			if cur == 0 {
				ready.V = append(ready.V, nd)
			}
		}

		self.Dict.SetStr("_in_degree_", inDeg)
		self.Dict.SetStr("_dependents_", dependents)
		self.Dict.SetStr("_ready_", ready)
		self.Dict.SetStr("_out_ready_", &object.List{})
		self.Dict.SetStr("_state_", object.NewInt(int64(tsStatePrepared)))
		self.Dict.SetStr("_n_done_", object.NewInt(0))
		self.Dict.SetStr("_n_total_", object.NewInt(int64(n)))
		return object.None, nil
	}})

	// is_active()
	tsCls.Dict.SetStr("is_active", &object.BuiltinFunc{Name: "is_active", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "is_active() requires self")
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "self must be an instance")
		}
		stateObj, _ := self.Dict.GetStr("_state_")
		stateInt, _ := toInt64(stateObj)
		if stateInt == int64(tsStateNew) {
			return nil, object.Errorf(i.valueErr, "prepare() must be called first")
		}

		nDoneObj, _ := self.Dict.GetStr("_n_done_")
		nTotalObj, _ := self.Dict.GetStr("_n_total_")
		nDone, _ := toInt64(nDoneObj)
		nTotal, _ := toInt64(nTotalObj)

		readyObj, _ := self.Dict.GetStr("_ready_")
		ready := readyObj.(*object.List)
		outReadyObj, _ := self.Dict.GetStr("_out_ready_")
		outReady := outReadyObj.(*object.List)

		// Active if not all done AND (something is ready or something is out for processing).
		active := nDone < nTotal && (len(ready.V) > 0 || len(outReady.V) > 0)
		return object.BoolOf(active), nil
	}})

	// __bool__ — same as is_active
	tsCls.Dict.SetStr("__bool__", &object.BuiltinFunc{Name: "__bool__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.False, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.False, nil
		}
		stateObj, _ := self.Dict.GetStr("_state_")
		stateInt, _ := toInt64(stateObj)
		if stateInt == int64(tsStateNew) {
			return object.False, nil
		}
		nDoneObj, _ := self.Dict.GetStr("_n_done_")
		nTotalObj, _ := self.Dict.GetStr("_n_total_")
		nDone, _ := toInt64(nDoneObj)
		nTotal, _ := toInt64(nTotalObj)
		readyObj, _ := self.Dict.GetStr("_ready_")
		ready := readyObj.(*object.List)
		outReadyObj, _ := self.Dict.GetStr("_out_ready_")
		outReady := outReadyObj.(*object.List)
		active := nDone < nTotal && (len(ready.V) > 0 || len(outReady.V) > 0)
		return object.BoolOf(active), nil
	}})

	// get_ready() -> tuple
	tsCls.Dict.SetStr("get_ready", &object.BuiltinFunc{Name: "get_ready", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "get_ready() requires self")
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "self must be an instance")
		}
		stateObj, _ := self.Dict.GetStr("_state_")
		stateInt, _ := toInt64(stateObj)
		if stateInt == int64(tsStateNew) {
			return nil, object.Errorf(i.valueErr, "prepare() must be called first")
		}
		self.Dict.SetStr("_state_", object.NewInt(int64(tsStateActive)))

		readyObj, _ := self.Dict.GetStr("_ready_")
		ready := readyObj.(*object.List)
		outReadyObj, _ := self.Dict.GetStr("_out_ready_")
		outReady := outReadyObj.(*object.List)

		result := make([]object.Object, len(ready.V))
		copy(result, ready.V)

		// Move ready → out_ready.
		outReady.V = append(outReady.V, ready.V...)
		ready.V = nil

		return &object.Tuple{V: result}, nil
	}})

	// done(*nodes)
	tsCls.Dict.SetStr("done", &object.BuiltinFunc{Name: "done", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "done() requires self")
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "self must be an instance")
		}
		stateObj, _ := self.Dict.GetStr("_state_")
		stateInt, _ := toInt64(stateObj)
		if stateInt == int64(tsStateNew) {
			return nil, object.Errorf(i.valueErr, "prepare() must be called first")
		}
		if stateInt == int64(tsStatePrepared) {
			return nil, object.Errorf(i.valueErr, "done() called before get_ready()")
		}

		nodes := a[1:]
		outReadyObj, _ := self.Dict.GetStr("_out_ready_")
		outReady := outReadyObj.(*object.List)
		inDegObj, _ := self.Dict.GetStr("_in_degree_")
		inDeg := inDegObj.(*object.Dict)
		dependentsObj, _ := self.Dict.GetStr("_dependents_")
		deps := dependentsObj.(*object.Dict)
		readyObj, _ := self.Dict.GetStr("_ready_")
		ready := readyObj.(*object.List)
		nDoneObj, _ := self.Dict.GetStr("_n_done_")
		nDone, _ := toInt64(nDoneObj)
		graphObj, _ := self.Dict.GetStr("_graph_")
		graphDict := graphObj.(*object.Dict)

		for _, nd := range nodes {
			// Validate: must be in graph.
			if _, found, _ := graphDict.Get(nd); !found {
				return nil, object.Errorf(i.valueErr, "node %s was not added to the graph", object.Repr(nd))
			}
			// Must be in out_ready.
			foundInOut := false
			newOutReady := make([]object.Object, 0, len(outReady.V))
			for _, orNode := range outReady.V {
				if eq, _ := object.Eq(orNode, nd); eq {
					if foundInOut {
						newOutReady = append(newOutReady, orNode)
					} else {
						foundInOut = true
					}
				} else {
					newOutReady = append(newOutReady, orNode)
				}
			}
			if !foundInOut {
				// Check if already done (not in out_ready and not in _ready_).
				return nil, object.Errorf(i.valueErr, "node %s was not returned by get_ready()", object.Repr(nd))
			}
			outReady.V = newOutReady
			nDone++

			// Decrement in-degree of dependents.
			depListObj, _, _ := deps.Get(nd)
			if depListObj != nil {
				depList := depListObj.(*object.List)
				for _, dep := range depList.V {
					curObj, _, _ := inDeg.Get(dep)
					cur, _ := toInt64(curObj)
					newCur := cur - 1
					inDeg.Set(dep, object.NewInt(newCur))
					if newCur == 0 {
						ready.V = append(ready.V, dep)
					}
				}
			}
		}

		self.Dict.SetStr("_n_done_", object.NewInt(nDone))
		return object.None, nil
	}})

	// static_order() -> iterator
	tsCls.Dict.SetStr("static_order", &object.BuiltinFunc{Name: "static_order", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "static_order() requires self")
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "self must be an instance")
		}

		// Call prepare() if not already prepared.
		stateObj, _ := self.Dict.GetStr("_state_")
		stateInt, _ := toInt64(stateObj)
		if stateInt == int64(tsStateNew) {
			if prepareFn, ok2 := tsCls.Dict.GetStr("prepare"); ok2 {
				if _, err := i.callObject(prepareFn, []object.Object{self}, nil); err != nil {
					return nil, err
				}
			}
		}

		// Collect all nodes in topological order by repeatedly calling get_ready/done.
		var order []object.Object
		getReadyFn, _ := tsCls.Dict.GetStr("get_ready")
		doneFn, _ := tsCls.Dict.GetStr("done")
		isActiveFn, _ := tsCls.Dict.GetStr("is_active")

		for {
			activeObj, err := i.callObject(isActiveFn, []object.Object{self}, nil)
			if err != nil {
				return nil, err
			}
			if !object.Truthy(activeObj) {
				break
			}
			batch, err := i.callObject(getReadyFn, []object.Object{self}, nil)
			if err != nil {
				return nil, err
			}
			batchTuple, ok2 := batch.(*object.Tuple)
			if !ok2 {
				break
			}
			if len(batchTuple.V) == 0 {
				break
			}
			for _, nd := range batchTuple.V {
				order = append(order, nd)
			}
			// Mark all batch nodes done.
			doneArgs := make([]object.Object, 0, len(batchTuple.V)+1)
			doneArgs = append(doneArgs, self)
			doneArgs = append(doneArgs, batchTuple.V...)
			if _, err := i.callObject(doneFn, doneArgs, nil); err != nil {
				return nil, err
			}
		}

		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(order) {
				return nil, false, nil
			}
			r := order[idx]
			idx++
			return r, true, nil
		}}, nil
	}})

	m.Dict.SetStr("TopologicalSorter", tsCls)
	return m
}
