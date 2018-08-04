package runxml

// nodeArena is a preallokated memory regions; to increase speed by preventing
// several sequential small allocations
type nodeArena []GenericNode

// Memory allocation parameter.  Start with STARTSIZE and increase by 2x
// until MAXSIZE
const (
	maxsize   int = 20000
	startsize int = 100
)

var currentSize = startsize

// get an GenericNode node from the arena
func (na *nodeArena) get() *GenericNode {
	// create new structs if empty
	if len(*na) == 0 {
		*na = make([]GenericNode, currentSize)
		currentSize *= 2
		currentSize = min(maxsize, currentSize)
	}
	n := &(*na)[len(*na)-1]
	*na = (*na)[:len(*na)-1]
	/*n := &(*na)[0] // possible optimization
	*na = (*na)[1:]*/
	return n
}

// attributeArena is a preallokated memory regions; to increase speed by preventing
// several sequential small allocations
type attributeArena []AttributeNode

var currAttrSize = startsize

// get an Attribute node from the arena
func (aa *attributeArena) get() *AttributeNode {
	//return &fake
	if len(*aa) == 0 {
		*aa = make([]AttributeNode, currAttrSize)
		currAttrSize *= 2
		currAttrSize = min(maxsize, currAttrSize)
	}
	n := &(*aa)[len(*aa)-1] // last elem
	*aa = (*aa)[:len(*aa)-1]
	//n := &(*aa)[0]
	//*aa = (*aa)[1:]
	return n
}
