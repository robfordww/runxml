package runxml

// NodeArena is a preallokated memory regions; to increase speed by preventing
// several sequential small allocations
type NodeArena []GenericNode

// Memory allocation parameter.  Start with STARTSIZE and increase by 2x
// until MAXSIZE
const (
	maxsize   int = 20000
	startsize int = 100
)

var currentSize = startsize

// Get an GenericNode node from the arena
func (na *NodeArena) Get() *GenericNode {
	//return &fakeg
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

// AttributeArena is a preallokated memory regions; to increase speed by preventing
// several sequential small allocations
type AttributeArena []AttributeNode

// Get an Attribute node from the arena
func (aa *AttributeArena) Get() *AttributeNode {
	//return &fake
	if len(*aa) == 0 {
		*aa = make([]AttributeNode, currentSize)
		currentSize *= 2
		currentSize = min(maxsize, currentSize)
	}
	n := &(*aa)[len(*aa)-1] // last elem
	*aa = (*aa)[:len(*aa)-1]
	//n := &(*aa)[0]
	//*aa = (*aa)[1:]
	return n
}
