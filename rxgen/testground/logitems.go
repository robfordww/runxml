package runxml

//runxml:logitem
//easyjson:json
type Logitem struct {
	Page   int           `json:"page"`
	Fruits []FruitDesc   `json:"fruits"`
	XIF    TestInterface `json:"iface"`
}

type TestInterface interface {
	IFFunc()
}

type FruitDesc struct {
	Name       string
	TastesGood bool
}

// <logitem>
// <id>62809477</id>
// <timestamp>2015-02-27T03:27:44Z</timestamp>
// <contributor>
//   <username>ClueBot NG</username>
//   <id>13286072</id>
// </contributor>
