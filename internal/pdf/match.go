package pdf

import (
	"log"
	"math"
	"reflect"
)

type MatchOptions struct {
	MatchReferences bool
	MatchDepth      bool
	MatchStream     bool
}

func MatchTypes(first ObjectType, second ObjectType, opts *MatchOptions) float64 {

	if reflect.TypeOf(first) != reflect.TypeOf(second) {
		return 0.0
	}

	switch first.(type) {
	case *Object:
		v1 := first.(*Object)
		v2 := second.(*Object)
		if opts.MatchDepth && v1.Depth != v2.Depth {
			return 0
		}
		if len(v1.Children) == 0 && len(v2.Children) == 0 {
			return 1
		}
		firstHasStream := false
		secondHasStream := false
		acc := 0.0
		for _, c1 := range v1.Children {
			if _, ok := c1.(*Stream); ok {
				firstHasStream = true
			}
			bestMatch := 0.0
			for _, c2 := range v2.Children {
				if _, ok := c2.(*Stream); ok {
					secondHasStream = true
				}
				score := MatchTypes(c1, c2, opts)
				if score > 0 && score > bestMatch {
					bestMatch = score
				}
				if score == 1 {
					break
				}
			}
			acc += bestMatch
		}
		if firstHasStream != secondHasStream {
			return 0
		}
		return acc / math.Max(float64(len(v1.Children)), float64(len(v2.Children)))
	case *Dictionary:
		v1 := first.(*Dictionary)
		v2 := second.(*Dictionary)
		if len(v1.Value) == 0 && len(v2.Value) == 0 {
			return 1
		}
		acc := 0.0
		marked := make(map[int]bool)
		for _, c1 := range v1.Value {
			bestMatch := 0.0
			bestKey := -1
			for key, c2 := range v2.Value {
				if marked[key] {
					continue
				}
				score := MatchTypes(&c1, &c2, opts)
				if score > 1 {
					log.Fatalln("score must be below or equal to 1")
				}
				if score > 0 && score > bestMatch {
					bestMatch = score
					bestKey = key
				}
				if score == 1 {
					break
				}
			}
			if bestKey != -1 {
				marked[bestKey] = true
				acc += bestMatch
			}
		}
		return acc / math.Max(float64(len(v1.Value)), float64(len(v2.Value)))
	case *Array:
		v1 := first.(*Array)
		v2 := second.(*Array)
		if len(v1.Value) == 0 && len(v2.Value) == 0 {
			return 1
		}
		acc := 0.0
		i := 0
		for _, c1 := range v1.Value {
			bestMatch := 0.0
			for j := i; j < len(v2.Value); j++ {
				score := MatchTypes(c1, v2.Value[j], opts)
				if score > 1 {
					log.Fatalln("score must be below or equal to 1")
				}
				if score > 0 && score > bestMatch {
					bestMatch = score
					i = j + 1
				}
				if score == 1 {
					break
				}
			}
			acc += bestMatch
		}
		return acc / math.Max(float64(len(v1.Value)), float64(len(v2.Value)))
	case *KeyValuePair:
		v1 := first.(*KeyValuePair)
		v2 := second.(*KeyValuePair)
		if v1.Key() != v2.Key() {
			return 0
		}
		_, ok1 := v1.V.(*Text)
		_, ok2 := v2.V.(*Text)
		if ok1 && ok2 && v1.Value() == v2.Value() {
			return 1
		}
		_, ok1 = v1.V.(*Number)
		_, ok2 = v2.V.(*Number)
		if ok1 && ok2 && v1.Value() == v2.Value() {
			return 1
		}
		_, ok1 = v1.V.(*Label)
		_, ok2 = v2.V.(*Label)
		if ok1 && ok2 && v1.Value() == v2.Value() {
			return 1
		}
		v := MatchTypes(v1.V, v2.V, opts)
		return v
	case *Text:
		v1 := first.(*Text)
		v2 := second.(*Text)
		if v1.Value == v2.Value {
			return 1.0
		}
		return 0
	case *Number:
		v1 := first.(*Number).Value
		v2 := second.(*Number).Value
		if v1 == v2 {
			return 1.0
		}
		if v1 == 0 || v2 == 0 {
			return 0
		}
		if v1 > 0 && v2 < 0 || v1 < 0 && v2 > 0 {
			return 0
		}
		if v1 < 0 {
			v1 = -v1
			v2 = -v2
		}
		return math.Min(v1, v2) / math.Max(v1, v2)
	case *Label:
		v1 := first.(*Label)
		v2 := second.(*Label)
		if v1.Value == v2.Value {
			return 1.0
		}
		return 0
	case *Boolean:
		v1 := first.(*Boolean)
		v2 := second.(*Boolean)
		if v1.Value == v2.Value {
			return 1.0
		}
		return 0.5
	case *Stream:
		return 1.0
	case *ObjectReference:
		if opts.MatchReferences {
			v1 := first.(*ObjectReference)
			v2 := second.(*ObjectReference)
			if v1.Link.Hash() == v2.Link.Hash() {
				return 1
			} else {
				return 0.5
			}
		} else {
			return 1.0
		}
	case *Null:
		if opts.MatchStream {
			v1 := first.(*Stream)
			v2 := second.(*Stream)
			if len(v1.Value) != len(v2.Value) {
				return 0
			}
			return 1.0
		}
		return 1.0
	case *HexString:
		return 1.0
	default:
		log.Fatalln("unhandled pdf type")
	}

	return 0.0
}
