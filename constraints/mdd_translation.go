package constraints

import (
	"errors"
	//	"fmt"
	"github.com/vale1410/bule/mdd"
	"github.com/vale1410/bule/sat"
	"math"
	"strconv"
)

func TranslateByMDD(pb *Threshold) (t ThresholdTranslation, err error) {
	t.Typ = ComplexMDD
	t.PB = pb
	pb.Normalize(AtMost, true)
	pb.Sort()
	// maybe do some more smart sorting?
	store := mdd.Init(len(pb.Entries))
	topId, _, _, err := CreateMDD(&store, pb.K, pb.Entries)
	store.Top = topId
	if err != nil {
		return t, err
	}
	t.Clauses = convertMDD2Clauses(store, pb)
	return t, err
}

func TranslateByMDDChain(pb *Threshold, chain Chain) (t ThresholdTranslation, err error) {

	//pb.Print10()
	//chain.Print()

	if len(chain) <= 1 {
		panic("Chain is smaller equal 1!!!")
	}

	t.Typ = ComplexMDDChain
	t.PB = pb

	store := mdd.Init(len(pb.Entries))
	topId, _, _, err := CreateMDDChain(&store, pb.K, pb.Entries, chain)
	store.Top = topId
	//store.Debug(true)

	if err != nil {
		return t, err
	}

	t.Clauses = convertMDDChainIntoClauses(store, pb)

	return
}

// TODO:optimize to remove 1 and 0 nodes in each level
// include some type of configuration
// Translate monotone MDDs to SAT
// If several children: assume literals in sequence of the PB
func convertMDD2Clauses(store mdd.MddStore, pb *Threshold) (clauses sat.ClauseSet) {

	pred := sat.Pred("mdd" + strconv.Itoa(pb.Id))

	top_lit := sat.Literal{true, sat.NewAtomP1(pred, store.Top)}
	clauses.AddTaggedClause("Top", top_lit)

	for _, n := range store.Nodes {
		v_id, l, vds := store.ClauseIds(*n)
		//fmt.Println(v_id, l, vds)
		if !n.IsZero() && !n.IsOne() {

			v_lit := sat.Literal{false, sat.NewAtomP1(pred, v_id)}
			for i, vd_id := range vds {
				vd_lit := sat.Literal{true, sat.NewAtomP1(pred, vd_id)}
				if i > 0 {
					literal := pb.Entries[len(pb.Entries)-l].Literal
					//if vd_id != 0 { // vd is not true
					clauses.AddTaggedClause("1B", v_lit, sat.Neg(literal), vd_lit)
					//} else {
					//	clauses.AddClause(sat.Neg(v_lit), sat.Neg(pb.Entries[len(pb.Entries)-l].Literal))
					//}
				} else {
					//if vd_id != 1 { // vd is not true
					clauses.AddTaggedClause("0B", v_lit, vd_lit)
					//}
				}
			}
		} else if n.IsZero() {
			v_lit := sat.Literal{false, sat.NewAtomP1(pred, v_id)}
			clauses.AddTaggedClause("False", v_lit)
		} else if n.IsOne() {
			v_lit := sat.Literal{true, sat.NewAtomP1(pred, v_id)}
			clauses.AddTaggedClause("True", v_lit)
		}
	}

	return
}

// Translate monotone MDDs to SAT
// Together with AMO translation
// TODO: complete implementation
func convertMDDChainIntoClauses(store mdd.MddStore, pb *Threshold) (clauses sat.ClauseSet) {

	pred := sat.Pred("mddc" + strconv.Itoa(pb.Id))

	top_lit := sat.Literal{true, sat.NewAtomP1(pred, store.Top)}
	clauses.AddTaggedClause("Top", top_lit)
	for _, n := range store.Nodes {
		v_id, l, vds := store.ClauseIds(*n)
		if !n.IsZero() && !n.IsOne() {

			v_lit := sat.Literal{false, sat.NewAtomP1(pred, v_id)}
			last_id := -1
			for i, vd_id := range vds {
				if last_id != vd_id {
					vd_lit := sat.Literal{true, sat.NewAtomP1(pred, vd_id)}
					if i > 0 {
						literal := pb.Entries[len(pb.Entries)-l+i-1].Literal
						clauses.AddTaggedClause("1B", v_lit, sat.Neg(literal), vd_lit)
					} else {
						clauses.AddTaggedClause("0B", v_lit, vd_lit)
					}
				}
				last_id = vd_id
			}
		} else if n.IsZero() {
			v_lit := sat.Literal{false, sat.NewAtomP1(pred, v_id)}
			clauses.AddTaggedClause("False", v_lit)
		} else if n.IsOne() {
			v_lit := sat.Literal{true, sat.NewAtomP1(pred, v_id)}
			clauses.AddTaggedClause("True", v_lit)
		}

	}

	return
}

// Chain: there are clauses  xi <-xi+1 <- xi+2 ... <- xi+k, and xi .. xi+k are in order of PB
// TODO: assumption: chain is in order with entries, starts somewhere
// TODO: chain is same polarity as entries, and coefficients in entries are ascending for chain
// TODO: extend to sequence of chains!!!
func CreateMDDChain(store *mdd.MddStore, K int64, entries []Entry, chain Chain) (int, int64, int64, error) {

	l := len(entries) ///level

	if store.MaxNodes < len(store.Nodes) {
		return 0, 0, 0, errors.New("mdd max nodes reached")
	}

	//chain.Print()
	//fmt.Println(l, K, entries)

	if id, wmin_cache, wmax_cache := store.GetByWeight(l, K); id != -1 {

		//	fmt.Println("exists", l, K, "[", wmin, wmax, "]")

		return id, wmin_cache, wmax_cache, nil

	} else {
		//domain of variable [0,1], extend to [0..n] soon (MDDs)
		// entry of variable domain, atom: Dom: 2

		var n mdd.Node
		var err error

		if len(chain) > 0 && chain[0] == entries[0].Literal { //chain mode
			var jumpEntries []Entry
			if len(entries) <= len(chain) {
				jumpEntries = []Entry{}
			} else {
				jumpEntries = entries[len(chain):]
			}
			// iterate over the chain
			n.Level = l
			n.Children = make([]int, len(chain)+1)

			n.Children[0], n.Wmin, n.Wmax, err = CreateMDDChain(store, K, jumpEntries, []sat.Literal{})

			if err != nil {
				return 0, 0, 0, err
			}

			acc := int64(0)

			//			fmt.Printf("entries:%v  chain: %v", entries, chain)
			for i, _ := range chain {

				if len(chain) > len(entries) || chain[i] != entries[i].Literal {
					panic("chain and PB are not aligned!!!! ")
				}

				var wmin2, wmax2 int64
				acc += entries[i].Weight
				n.Children[i+1], wmin2, wmax2, err = CreateMDDChain(store, K-acc, jumpEntries, chain)
				n.Wmin = maxx(n.Wmin, wmin2+acc)
				n.Wmax = min(n.Wmax, wmax2+acc)

				if err != nil {
					return 0, 0, 0, err
				}

			}

		} else { //usual mode
			dom := 2
			n.Level = l
			n.Children = make([]int, dom)
			n.Wmin = math.MinInt64
			n.Wmax = math.MaxInt64

			var err error
			for i := int64(0); i < int64(dom); i++ {
				var wmin2, wmax2 int64

				n.Children[i], wmin2, wmax2, err = CreateMDDChain(store, K-i*entries[0].Weight, entries[1:], chain)

				n.Wmin = maxx(n.Wmin, wmin2+i*entries[0].Weight)
				n.Wmax = min(n.Wmax, wmax2+i*entries[0].Weight)

				if err != nil {
					return 0, 0, 0, err
				}
			}
		}

		return store.Insert(n), n.Wmin, n.Wmax, nil
	}
}

func CreateMDD(store *mdd.MddStore, K int64, entries []Entry) (int, int64, int64, error) {

	l := len(entries) ///level

	if store.MaxNodes < len(store.Nodes) {
		return 0, 0, 0, errors.New("mdd max nodes reached")
	}

	//fmt.Println(l, K, entries)

	if id, wmin_cache, wmax_cache := store.GetByWeight(l, K); id != -1 {

		//	fmt.Println("exists", l, K, "[", wmin, wmax, "]")
		return id, wmin_cache, wmax_cache, nil

	} else {
		//domain of variable [0,1], extend to [0..n] soon (MDDs)
		// entry of variable domain, atom: Dom: 2

		dom := 2

		var n mdd.Node
		n.Level = l
		n.Children = make([]int, dom)
		n.Wmin = math.MinInt64
		n.Wmax = math.MaxInt64

		var err error
		for i := int64(0); i < int64(dom); i++ {
			var wmin2, wmax2 int64
			n.Children[i], wmin2, wmax2, err = CreateMDD(store, K-i*entries[0].Weight, entries[1:])
			n.Wmin = maxx(n.Wmin, wmin2+i*entries[0].Weight)
			n.Wmax = min(n.Wmax, wmax2+i*entries[0].Weight)

			if err != nil {
				return 0, 0, 0, err
			}

			//			}
		}

		return store.Insert(n), n.Wmin, n.Wmax, nil
	}
}

func min(a, b int64) int64 {
	if a <= b {
		return a
	} else {
		return b
	}
}

func maxx(a, b int64) int64 {
	if a >= b {
		return a
	} else {
		return b
	}
}