package constraints

import (
	"fmt"
	"math"
	"os"
	//"sort"
	"strconv"
)

func (t *Threshold) Print2() {
	fmt.Println(t.Desc)

	first := true
	for _, x := range t.Entries {
		l := x.Literal
		if !first {
			fmt.Printf("+ ")
		}
		first = false

		fmt.Print(BinaryStr(x.Weight))
		fmt.Print(x.Weight)
		fmt.Print(l.ToTex())
	}
	switch t.Typ {
	case AtMost:
		fmt.Print(" <= ")
	case AtLeast:
		fmt.Print(" >= ")
	case Equal:
		fmt.Print(" == ")
	}

	fmt.Print(BinaryStr(t.K))

	fmt.Println()
	fmt.Println()
}

func (t *Threshold) WriteFormula(base int, file *os.File) {

	var w string

	file.Write([]byte("$$"))
	for _, x := range t.Entries {
		if base == 2 {
			w = BinaryStr(x.Weight)
		} else if base == 10 {
			w = fmt.Sprintf("%v", x.Weight)
		} else {
			panic("only base 2 and 10 supported")
		}
		if x.Weight < 0 {
			file.Write([]byte(fmt.Sprintf("%v \\cdot %v ", w, x.Literal.ToTex())))
		} else {
			file.Write([]byte(fmt.Sprintf("+%v \\cdot %v ", w, x.Literal.ToTex())))
		}
	}
	switch t.Typ {
	case AtMost:
		file.Write([]byte(" \\leq "))
	case AtLeast:
		file.Write([]byte(" \\geq "))
	case Equal:
		file.Write([]byte(" = "))
	}
	if base == 2 {
		w = BinaryStr(t.K)
	} else if base == 10 {
		w = fmt.Sprintf("%v", t.K)
	} else {
		panic("only base 2 and 10 supported")
	}
	file.Write([]byte(fmt.Sprintf("%v $$\n", w)))
}

func (t *Threshold) PrintGurobi() {
	for _, x := range t.Entries {
		l := x.Literal

		if x.Weight > 0 {
			fmt.Printf(" +")
		}

		if x.Weight != 1 {
			fmt.Printf(" ")
			fmt.Print(x.Weight)
		}
		fmt.Print(l.ToTxt())
	}
	switch t.Typ {
	case AtMost:
		fmt.Print(" <= ")
	case AtLeast:
		fmt.Print(" >= ")
	case Equal:
		fmt.Print(" = ")
	}
	fmt.Println(t.K)

}

func (t *Threshold) PrintPBO() {
	for _, x := range t.Entries {
		l := x.Literal

		if x.Weight > 0 {
			fmt.Printf("+")
		}
		fmt.Print(x.Weight, " ", l.ToPBO(), " ")
	}
	switch t.Typ {
	case AtMost:
		fmt.Print("<= ")
	case AtLeast:
		fmt.Print(">= ")
	case Equal:
		fmt.Print("== ")
	}
	fmt.Println(t.K, ";")
}

func (t *Threshold) Print10() {
	for _, x := range t.Entries {
		l := x.Literal

		if x.Weight > 0 {
			fmt.Printf(" +")
		}
		if x.Weight == 1 {
			fmt.Print(l.ToTxt())
		} else {
			fmt.Print(" ", x.Weight, l.ToTxt())
		}
	}
	switch t.Typ {
	case AtMost:
		fmt.Print(" <= ")
	case AtLeast:
		fmt.Print(" >= ")
	case Equal:
		fmt.Print(" == ")
	}
	fmt.Println(t.K, ";")

}

func (t *Threshold) PrintGringo() {

	if len(t.Entries) > 0 {

		switch t.Typ {
		case AtMost:
			fmt.Print(":- ", t.K+1, " [ ")
		case AtLeast:
			fmt.Print(":- [ ")
		case Equal:
			fmt.Print(":- not ", t.K, " [ ")
		case Optimization:
			fmt.Print("#minimize[")
		}

		for i, x := range t.Entries {
			if i != 0 {
				fmt.Print(" , ")
			}
			if x.Weight != 1 {
				fmt.Print(x.Literal.ToTxt(), "=", x.Weight)
			} else {
				fmt.Print(x.Literal.ToTxt())
			}
		}

		switch t.Typ {
		case AtMost:
			fmt.Print(" ]")
		case AtLeast:
			fmt.Print(" ] ", t.K-1)
		case Equal:
			fmt.Print(" ] ", t.K)
		case Optimization:
			fmt.Print("]")
		}
		fmt.Println(".")
	}

}

func BinaryStr(n int64) (s string) {
	bin := Binary(n)
	for i := len(bin) - 1; i >= 0; i-- {
		s += strconv.Itoa(bin[i])
	}
	return
}

// binary
// 23 = 10111
// special case if n==0 then return empty slice
// panic if n<0
func Binary(n int64) (bin []int) {

	var s int64

	if n < 0 {
		panic("binary representation of number smaller than 0")
	} else if n == 0 {
		s = 0
	} else {
		s = int64(math.Logb(float64(n))) + 1
	}

	bin = make([]int, s)

	i := 0
	var m int64

	for n != 0 {
		m = n / 2
		//fmt.Println(i, n, m)
		if n != m*2 {
			bin[i] = 1
		}
		n = m
		i++
	}
	return
}