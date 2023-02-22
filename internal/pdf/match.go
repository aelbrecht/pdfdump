package pdf

import (
	"log"
	"math"
	"reflect"
)

func MatchTypes(first ObjectType, second ObjectType) float64 {

	if reflect.TypeOf(first) != reflect.TypeOf(second) {
		return 0.0
	}

	switch first.(type) {
	case *Object:
		v1 := first.(*Object)
		v2 := second.(*Object)
		acc := 0.0
		for _, c1 := range v1.Children {
			bestMatch := 0.0
			for _, c2 := range v2.Children {
				score := MatchTypes(c1, c2)
				if score > 0 && score > bestMatch {
					bestMatch = score
				}
				if score == 1 {
					break
				}
			}
			acc += bestMatch
		}
		return acc / math.Max(float64(len(v1.Children)), float64(len(v2.Children)))
	case *Dictionary:
		v1 := first.(*Dictionary)
		v2 := second.(*Dictionary)
		acc := 0.0
		for _, c1 := range v1.Value {
			bestMatch := 0.0
			for _, c2 := range v2.Value {
				if c1.Key() != c2.Key() {
					continue
				}
				score := MatchTypes(c1.V, c2.V)
				if score > 0 && score > bestMatch {
					bestMatch = score
				}
				if score == 1 {
					break
				}
			}
			acc += bestMatch
		}
		return acc / math.Max(float64(len(v1.Value)), float64(len(v2.Value)))
	case *Array:
		v1 := first.(*Array)
		v2 := second.(*Array)
		acc := 0.0
		for _, c1 := range v1.Value {
			bestMatch := 0.0
			for _, c2 := range v2.Value {
				score := MatchTypes(c1, c2)
				if score > 0 && score > bestMatch {
					bestMatch = score
				}
				if score == 1 {
					break
				}
			}
			acc += bestMatch
		}
		return acc / math.Max(float64(len(v1.Value)), float64(len(v2.Value)))
	case *String:
		v1 := first.(*String)
		v2 := second.(*String)
		if v1.Value == v2.Value {
			return 1.0
		}
		return 0
	case *FloatingNumber:
		v1 := first.(*FloatingNumber).Value
		v2 := second.(*FloatingNumber).Value
		if v1 == v2 {
			return math.Min(v1, v2) / math.Max(v1, v2)
		}
		return 0
	case *IntegerNumber:
		v1 := float64(first.(*IntegerNumber).Value)
		v2 := float64(second.(*IntegerNumber).Value)
		if v1 == v2 {
			return math.Min(v1, v2) / math.Max(v1, v2)
		}
		return 0
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
		return 1.0
	case *Null:
		return 1.0
	default:
		log.Fatalln("unhandled pdf type")
	}

	return 0.0
}
