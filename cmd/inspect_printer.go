package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MontFerret/ferret/v2/pkg/bytecode"
)

func printSummary(p *bytecode.Program) {
	name := "<anonymous>"

	if p.Source != nil {
		name = p.Source.Name()
	}

	totalFuncs := len(p.Functions.Host) + len(p.Functions.UserDefined)

	fmt.Printf("Source: %s\n", name)
	fmt.Printf("Functions: %d\n", totalFuncs)
	fmt.Printf("Constants: %d\n", len(p.Constants))
	fmt.Printf("Registers: %d\n", p.Registers)
	fmt.Printf("Instructions: %d\n", len(p.Bytecode))
	fmt.Printf("Catch blocks: %d\n", len(p.CatchTable))
	fmt.Printf("ISA version: %d\n", p.ISAVersion)

	if len(p.Params) > 0 {
		fmt.Printf("Entry params: %s\n", strings.Join(p.Params, ", "))
	} else {
		fmt.Println("Entry params: (none)")
	}

	if len(p.Functions.UserDefined) > 0 {
		udfs := make([]bytecode.UDF, len(p.Functions.UserDefined))
		copy(udfs, p.Functions.UserDefined)

		sort.Slice(udfs, func(i, j int) bool {
			return udfs[i].Entry < udfs[j].Entry
		})

		mainIns := len(p.Bytecode)

		if len(udfs) > 0 {
			mainIns = udfs[0].Entry
		}

		fmt.Println("Functions:")
		fmt.Printf("  - main(params=%d, regs=%d, ins=%d)\n", len(p.Params), p.Registers, mainIns)

		for i, udf := range udfs {
			ins := len(p.Bytecode) - udf.Entry

			if i+1 < len(udfs) {
				ins = udfs[i+1].Entry - udf.Entry
			}

			fmt.Printf("  - %s(params=%d, regs=%d, ins=%d)\n", udf.Name, udf.Params, udf.Registers, ins)
		}
	}
}

func printBytecode(p *bytecode.Program) {
	for ip, instr := range p.Bytecode {
		if label, ok := p.Metadata.Labels[ip]; ok {
			fmt.Printf("%s:\n", label)
		}

		ops := make([]string, 0, 3)

		for _, op := range instr.Operands {
			if op != bytecode.NoopOperand {
				ops = append(ops, op.String())
			}
		}

		if len(ops) > 0 {
			fmt.Printf("  %4d: %-12s %s\n", ip, instr.Opcode, strings.Join(ops, " "))
		} else {
			fmt.Printf("  %4d: %s\n", ip, instr.Opcode)
		}
	}
}

func printConstants(p *bytecode.Program) {
	for i, c := range p.Constants {
		fmt.Printf("  C%-4d %s\n", i, c.String())
	}
}

func printFunctions(p *bytecode.Program) {
	if len(p.Functions.Host) > 0 {
		fmt.Println("Host functions:")

		names := make([]string, 0, len(p.Functions.Host))

		for name := range p.Functions.Host {
			names = append(names, name)
		}

		sort.Strings(names)

		for _, name := range names {
			fmt.Printf("  %s(params=%d)\n", name, p.Functions.Host[name])
		}
	}

	if len(p.Functions.UserDefined) > 0 {
		if len(p.Functions.Host) > 0 {
			fmt.Println()
		}

		fmt.Println("User-defined functions:")

		for _, udf := range p.Functions.UserDefined {
			fmt.Printf("  %s(params=%d, regs=%d, entry=%d)\n", udf.Name, udf.Params, udf.Registers, udf.Entry)
		}
	}
}

func printSpans(p *bytecode.Program) {
	if len(p.Metadata.DebugSpans) == 0 {
		fmt.Println("  (no debug spans)")

		return
	}

	for ip, span := range p.Metadata.DebugSpans {
		if span.Start <= 0 && span.End <= 0 {
			continue
		}

		fmt.Printf("  %4d: [%d, %d)\n", ip, span.Start, span.End)
	}
}
