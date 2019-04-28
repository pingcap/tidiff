package directive

import (
	"math/rand"
	"reflect"
)

var Functions = map[string]interface{}{
	"count":   count,
	"first":   first,
	"head":    head,
	"tail":    tail,
	"int":     randint,
	"char":    randnstring,
	"varchar": randstring,
}

func count(n int) []struct{} {
	return make([]struct{}, n)
}

func first(x int) bool {
	return x == 0
}

func last(x int, a interface{}) bool {
	return x == reflect.ValueOf(a).Len()-1
}

func head(x int, a interface{}) bool {
	return x < reflect.ValueOf(a).Len()-1
}

func tail(x int) bool {
	return x > 0
}

func randint(min, rg int) int {
	return min + rand.Intn(rg)
}

func randnstring(n int) string {
	if n <= 0 {
		return ""
	}
	result := make([]byte, n)
	for i := range result {
		result[i] = 'A' + byte(rand.Intn(int('z'-'A')))
	}
	return string(result)
}

func randstring(maxn int) string {
	return randnstring(maxn/2 + rand.Intn(maxn/2))
}
