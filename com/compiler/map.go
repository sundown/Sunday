package compiler

import (
	"sundown/solution/temporal"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

func (state *State) CompileInlineMap(app *temporal.Application) value.Value {
	if app.Argument.TypeOf.Tuple == nil {
		panic("Map requires Tuple")
	}

	// Return type of function to be mapped
	f_returns := app.Argument.TypeOf.Tuple[0]

	// The vector in AST
	vec := app.Argument.Atom.Tuple[1]

	// The vector in LLVM
	llvec := state.CompileExpression(vec)

	head_type := vec.TypeOf.AsLLType()
	elm_type := vec.TypeOf.Vector.AsLLType()
	to_head_type := f_returns.AsVector().AsLLType()
	to_elm_type := f_returns.AsLLType()

	should_store := true
	if f_returns.Equals(temporal.AtomicType("Void")) {
		should_store = false
	}

	// Map is 1:1 so leng and cap are just copied from input vector
	leng := state.Block.NewGetElementPtr(head_type, llvec, I32(0), vectorLenOffset)
	cap := state.Block.NewGetElementPtr(head_type, llvec, I32(0), vectorCapOffset)

	var head *ir.InstAlloca
	var body *ir.InstBitCast

	if should_store {
		// Copy length
		state.Block.NewStore(
			state.Block.NewLoad(types.I64, leng),
			state.Block.NewGetElementPtr(head_type, llvec, I32(0), vectorLenOffset))

		// Copy capacity
		state.Block.NewStore(
			state.Block.NewLoad(types.I64, cap),
			state.Block.NewGetElementPtr(head_type, llvec, I32(0), vectorCapOffset))
		// Allocate a body of capacity * element width, and cast to element type
		body = state.Block.NewBitCast(
			state.Block.NewCall(state.GetCalloc(),
				I32(vec.Atom.Vector[0].TypeOf.WidthInBytes()),                         // Byte size of elements
				state.Block.NewTrunc(state.Block.NewLoad(types.I64, cap), types.I32)), // How much memory to alloc
			types.NewPointer(to_elm_type)) // Cast alloc'd memory to typ

		head = state.Block.NewAlloca(to_head_type)
	}

	// -------------
	// # LOOP BODY #
	// -------------
	if app.Argument.Atom.Tuple[1].TypeOf.Vector != nil {
		vec_body := state.Block.NewLoad(
			types.NewPointer(elm_type),
			state.Block.NewGetElementPtr(head_type, llvec, I32(0), vectorBodyOffset))

		counter := state.Block.NewAlloca(types.I64)
		state.Block.NewStore(I64(0), counter)

		// Body
		// Get elem, add to accum, increment counter, conditional jump to body
		loopblock := state.CurrentFunction.NewBlock("")
		state.Block.NewBr(loopblock)
		state.Block = loopblock
		// Add to accum
		cur_counter := loopblock.NewLoad(types.I64, counter)

		var cur_elm value.Value = loopblock.NewGetElementPtr(elm_type, vec_body, cur_counter)

		if vec.TypeOf.Vector.Atomic != nil {
			cur_elm = loopblock.NewLoad(elm_type, cur_elm)
		}

		var call value.Value

		if app.Argument.Atom.Tuple[0].Atom.Function.Special {
			call = state.GetSpecialCallable(app.Argument.Atom.Tuple[0].Atom.Function.Ident)(vec.TypeOf.Vector, cur_elm)
		} else {
			call = loopblock.NewCall(
				state.CompileExpression(app.Argument.Atom.Tuple[0]),
				cur_elm)

		}

		if should_store {
			loopblock.NewStore(
				call,
				loopblock.NewGetElementPtr(to_elm_type, body, cur_counter))
		}

		// Increment counter
		loopblock.NewStore(
			loopblock.NewAdd(
				loopblock.NewLoad(types.I64, counter),
				I64(1)),
			counter)

		// Possibly change load to another add or something, probably expensive
		cond := loopblock.NewICmp(enum.IPredSLT,
			loopblock.NewLoad(types.I64, counter),
			loopblock.NewLoad(types.I64, leng))

		exitblock := state.CurrentFunction.NewBlock("")
		loopblock.NewCondBr(cond, loopblock, exitblock)
		state.Block = exitblock

		if should_store {
			state.Block.NewStore(body,
				state.Block.NewGetElementPtr(to_head_type, head, I32(0), vectorBodyOffset))
			return head
		} else {
			return nil
		}

	} else {
		panic("Map needs (F, [T])")
	}
}
