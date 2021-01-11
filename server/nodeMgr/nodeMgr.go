package nodeMgr

import "sync"

type NodeMgr struct {
	//即将分配的ID
	preAllocateID uint32
	//保护nodeMap
	mu sync.RWMutex
	nodeMap map[uint32]*Node
}

func NewNodeMgr()*NodeMgr{
	return &NodeMgr{
		nodeMap: make(map[uint32]*Node),
	}
}

func (m *NodeMgr) NodesNum()int{
	return len(m.nodeMap)
}

func (m *NodeMgr) GetNode(id uint32) (node *Node,ok bool){
	m.mu.RLock()
	defer m.mu.RUnlock()
	node,ok=m.nodeMap[id]
	return
}

func (m *NodeMgr) AddNode(node *Node){
	m.mu.Lock()
	defer m.mu.Unlock()
	m.preAllocateID++
	node.id=m.preAllocateID
	m.nodeMap[node.id]=node
}

func (m *NodeMgr)DeleteNode(node *Node){
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.nodeMap,node.id)
}
