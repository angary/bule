package constraints

import (
	"math"
	"sort"
	"strconv"

	"github.com/vale1410/bule/glob"
	"github.com/vale1410/bule/sat"
)

type EquationType int

const (
	LE  EquationType = iota //"<="
	GE                      //">="
	EQ                      //"=="
	OPT                     //"MAX"
)

func (e EquationType) String() string {
	switch e {
	case LE:
		return "<="
	case GE:
		return ">="
	case EQ:
		return "=="
	case OPT:
		return "MAX"
	}
	return ""
}

type Entry struct {
	Literal sat.Literal
	Weight  int64
}

type Threshold struct {
	Id         int // unique id to reference Threshold in encodings
	Entries    []Entry
	K          int64
	Typ        EquationType
	Translated bool //indicates if constraint is translated
	TransTyp   TranslationType
	Clauses    sat.ClauseSet
	Chains     Chains // has to be in order of Entries
	Err        error  // some error in the translation
}

// creates copy of pb, new allocation of Entry slice
// copy empties the clauseSet!
func (pb *Threshold) Copy() (pb2 Threshold) {
	pb2 = *pb
	pb2.Entries = make([]Entry, len(pb.Entries))
	copy(pb2.Entries, pb.Entries)
	pb2.Clauses = sat.ClauseSet{}
	//pb2.Print10()
	//fmt.Println(pb2.Chains)
	return
}

// returns the encoding of this PB
func (pb *Threshold) Translate(K int64) sat.ClauseSet {
	pb_K := pb.Copy() //removes all clauses !
	pb_K.K = K
	pb_K.Typ = LE
	//glob.D("# of chains", len(pb_K.Chains))
	if len(pb_K.Chains) > 0 {
		pb_K.TranslateByMDDChain(pb_K.Chains)
	} else {
		pb_K.Categorize1()
	}

	if pb_K.Err != nil { // case MDD construction did go wrong!
		glob.D("Capacity of MDD reached, trying to solve by not taking chains into account")
		pb_K := pb.Copy() //removes all clauses !
		pb_K.K = K
		pb_K.Typ = LE
		pb_K.Categorize1()
	}
	return pb_K.Clauses
}

// returns the encoding of this PB
func (pb *Threshold) RewriteSameWeights() {

	// go to the end of chains
	// reorder PB after this descending
	// group variables according to weights
	// re-encode same weights by a sorter (use createSorter, createEncoding)
	// generate Chains for this

	return
}

func (pb *Threshold) Evaluate(a sat.Assignment) (r int64) {

	for _, e := range pb.Entries {
		v, b := a[e.Literal.A.Id()]
		glob.A(b, "Literal not found in assignment: ", e.Literal.ToTxt())
		if e.Literal.Sign {
			r += int64(v) * e.Weight
		} else {
			r += (1 - int64(v)) * e.Weight
		}
	}

	return r
}

func (pb *Threshold) Empty() bool {
	return len(pb.Entries) == 0
}

func (pb *Threshold) Positive() bool {
	for _, x := range pb.Entries {
		if x.Weight < 0 {
			return false
		}
	}
	return true
}

func (pb *Threshold) IdS() string {
	return strconv.Itoa(pb.Id)
}

type Chain []sat.Literal
type Chains []Chain
type Lits []sat.Literal

// creates an AtMost constraint
// with coefficients in weights,
// variables x1..xm
func CreatePB(weights []int64, K int64) (pb Threshold) {
	return CreatePBOffset(0, weights, K)
}

// creates an AtMost constraint
// with coefficients in weights,
// variables x1..xm
func CreatePBOffset(offset int, weights []int64, K int64) (pb Threshold) {

	pb.Entries = make([]Entry, len(weights))
	pb.Typ = LE
	pb.K = K

	p := sat.Pred("x")
	for i := 0; i < len(weights); i++ {
		l := sat.Literal{true, sat.NewAtomP1(p, i+offset)}
		pb.Entries[i] = Entry{l, weights[i]}
	}
	return
}

// finds trivially implied facts, returns set of facts
// removes such entries from the pb
// threshold can become empty!
func (t *Threshold) RemoveZeros() {

	entries := make([]Entry, len(t.Entries))
	copy(entries, t.Entries)

	// alternative faster implementation that does not
	// keeps order
	j := 0
	for _, x := range t.Entries {
		if x.Weight != 0 {
			entries[j] = x
			j++
		}
	}
	t.Entries = entries[:j]
}

func (t *Threshold) Literals() (lits []sat.Literal) {
	lits = make([]sat.Literal, len(t.Entries))
	for i, x := range t.Entries {
		lits[i] = x.Literal
	}
	return
}

func (pb *Threshold) GetEntriesAfterChains() []Entry {
	current := 0

	for _, chain := range pb.Chains {
		for _, lit := range chain {
			glob.A(pb.Entries[current].Literal == lit, "chain is not aligned with PB", chain, pb)
			current++
		}
	}
	return pb.Entries[current:]
}

// finds trivially implied facts, returns set of facts
// removes such entries from the pb
// threshold can become empty!
func (pb *Threshold) Simplify() {

	if pb.Typ == OPT {
		glob.D(pb.IdS(), " is not simplyfied because is OPT")
		return
	}

	pb.Normalize(LE, true)

	entries := make([]Entry, 0, len(pb.Entries))

	for _, x := range pb.Entries {
		if x.Weight > pb.K {
			pb.Clauses.AddTaggedClause(pb.IdS()+"-simpl", sat.Neg(x.Literal))
		} else {
			entries = append(entries, x)
		}
	}
	pb.Entries = entries
	pb.Normalize(GE, true)

	if pb.SumWeights() == pb.K {
		for _, x := range pb.Entries {
			pb.Clauses.AddTaggedClause("Fact", x.Literal)
		}
		pb.Entries = []Entry{}
		pb.K = 0
	}

	return
}

// all weights are the same; performs rounding
// if this is true, then all weights are 1, and K is the cardinality
func (t *Threshold) Cardinality() (allSame bool, literals []sat.Literal) {
	glob.A(len(t.Chains) == 0, "cant reorder Entries with chains")

	t.NormalizePositiveCoefficients()
	allSame = true

	coeff := t.Entries[0].Weight
	for _, x := range t.Entries {
		if x.Weight != coeff {
			allSame = false
			break
		}
	}

	if allSame {
		literals = make([]sat.Literal, len(t.Entries))
		t.K = int64(math.Ceil(float64(t.K) / float64(coeff)))
		for i, x := range t.Entries {
			t.Entries[i].Weight = 1
			literals[i] = x.Literal
		}

	}

	return allSame, literals
}

func (t *Threshold) NormalizePositiveCoefficients() {
	glob.A(len(t.Chains) == 0, "cant reorder Entries with chains")

	for i, e := range t.Entries {
		if t.Entries[i].Weight < 0 {
			t.Entries[i].Literal = sat.Neg(e.Literal)
			t.K -= t.Entries[i].Weight
			t.Entries[i].Weight = -t.Entries[i].Weight
		}
	}
}

func (t *Threshold) NormalizePositiveLiterals() {
	glob.A(len(t.Chains) == 0, "cant reorder Entries with chains")

	for i, e := range t.Entries {
		if t.Entries[i].Literal.Sign == false {
			t.Entries[i].Literal = sat.Neg(e.Literal)
			t.K -= t.Entries[i].Weight
			t.Entries[i].Weight = -t.Entries[i].Weight
		}
	}
}

func (t *Threshold) Multiply(c int64) {
	if c == 0 {
		panic("multiplyer is 0")
	}
	for i, e := range t.Entries {
		t.Entries[i].Weight = c * e.Weight
	}

	t.K = c * t.K

	if c < 0 {
		switch t.Typ {
		case LE:
			t.Typ = GE
		case GE:
			t.Typ = LE
		default:
			//nothing
		}
	}
}

// normalizes the threshold
// Change EquationType in case of LE/GE
// in case of EQ and OPT, positive weights
func (t *Threshold) Normalize(typ EquationType, posWeights bool) {
	glob.A(len(t.Chains) == 0, "cant reorder Entries with chains")

	if (typ == LE && t.Typ == GE) || (typ == GE && t.Typ == LE) {
		t.Multiply(-1)
	}

	if posWeights {
		t.NormalizePositiveCoefficients()
	} else {
		t.NormalizePositiveLiterals()
	}

	return
}

// finds the subexpression of chain1 in e and
// returns the entries of chain1 existing in e.
func CleanChain(entries []Entry, chain1 Chain) (chain2 Chain) {
	glob.A(len(chain1) > 0, "no non-empty chains")

	chain2 = make(Chain, len(chain1))

	e := 0
	for i, x := range entries {
		if x.Literal == chain1[0] {
			e = i
			break
		}
		glob.A(i <= len(entries)-1, "chain must exist within entries")
	}

	j2 := 0
	for j1, l := range chain1 {
		//fmt.Println("e", e, "j1", j1, "j2", j2)
		if e+j2 == len(entries) {
			break
		}
		if l == entries[e+j2].Literal {
			chain2[j2] = chain1[j1]
			j2++
		}
	}

	return chain2[:j2]
}

// assumption is that pb2 is already a subsequece of pb1
// TODO deprecated
func CommonSlice(e1 []Entry, e2 []Entry) (bool, []Entry) {
	for i, x := range e1 {
		if x.Literal == e2[0].Literal {
			return true, e1[i : i+len(e2)]
		}
	}
	return false, []Entry{}
}

// assumption is that pb2 is already a subsequece of pb1
func PositionSlice(e1 []Entry, e2 []Entry) (bool, []int) {
	//find min coefficient, to subtract
	pos := make([]int, len(e2))

	j := 0
	for i, x := range e1 {
		if j == len(pos) {
			break
		}
		if x.Literal == e2[j].Literal {
			pos[j] = i
			j++
		}
	}
	if j != len(pos) {
		return false, []int{}
	}
	return false, pos
}

// sums up all weights
func (t *Threshold) SumWeights() (total int64) {
	for _, e := range t.Entries {
		total += e.Weight
	}
	return
}

func (t *Threshold) SortVar() {
	sort.Sort(EntriesVariables(t.Entries))
}

func (t *Threshold) SortAscending() {
	sort.Sort(EntriesAscending(t.Entries))
}

func (t *Threshold) SortDescending() {
	sort.Sort(EntriesDescending(t.Entries))
}

type EntriesVariables []Entry
type EntriesAscending []Entry
type EntriesDescending []Entry

func (a EntriesVariables) Len() int      { return len(a) }
func (a EntriesVariables) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a EntriesVariables) Less(i, j int) bool {
	return a[i].Literal.A.Id() <= a[j].Literal.A.Id()
}

func (a EntriesDescending) Len() int           { return len(a) }
func (a EntriesDescending) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a EntriesDescending) Less(i, j int) bool { return a[i].Weight >= a[j].Weight }

func (a EntriesAscending) Len() int           { return len(a) }
func (a EntriesAscending) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a EntriesAscending) Less(i, j int) bool { return a[i].Weight <= a[j].Weight }
