package command

import (
	"BackendTemplate/pkg/utils"
	"strings"
	"sync"
)

type FileBrowserQueue struct {
	mutex  sync.Mutex
	Queues map[string]chan string
}

var VarFileBrowserQueue = &FileBrowserQueue{Queues: make(map[string]chan string)}

func (q *FileBrowserQueue) Add(uid string, files string) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if _, exists := q.Queues[uid]; !exists {
		q.Queues[uid] = make(chan string, 1)
	}
	select {
	case <-q.Queues[uid]: // 清空旧数据
	default: // 若通道为空，继续发送
	}

	// 发送最新的 pids 数据
	q.Queues[uid] <- files
}

func (q *FileBrowserQueue) GetOrCreateQueue(uid string) chan string {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if _, exists := q.Queues[uid]; !exists {
		q.Queues[uid] = make(chan string, 1) // 带缓冲区的通道，防止阻塞
	}
	return q.Queues[uid]
}

type FileNode struct {
	Name         string      `json:"name"`
	Size         string      `json:"size"`
	Type         string      `json:"type"` // "D" 表示目录，"F" 表示文件
	Path         string      `json:"path"`
	ModifiedTime string      `json:"modifiedTime,omitempty"`
	Children     []*FileNode `json:"children,omitempty"`
}

var UidFileBrowser = make(map[string][]*FileNode)
var fileBrowserMutex sync.Mutex

func ParseDirectoryString(uid string, data string) []*FileNode {
	fileBrowserMutex.Lock()
	defer fileBrowserMutex.Unlock()

	lines := strings.Split(data, "\n")
	if len(lines) < 4 {
		return UidFileBrowser[uid]
	}

	// 获取当前目录路径
	currentDir := strings.TrimSuffix(lines[0], "/*")
	currentDir = strings.Replace(currentDir, "\\", "/", -1)
	currentDir = strings.TrimSuffix(currentDir, "/")

	// 判断操作系统类型
	isWindows := len(currentDir) >= 2 && currentDir[1] == ':'

	// 获取盘符/根节点名称
	var rootName string
	if isWindows {
		rootName = currentDir[:2] // "C:", "Z:" 等
	} else {
		rootName = "/"
	}

	// 初始化或获取根节点
	if _, exists := UidFileBrowser[uid]; !exists {
		UidFileBrowser[uid] = []*FileNode{{
			Name:     rootName,
			Type:     "D",
			Path:     rootName,
			Children: []*FileNode{},
		}}
	}

	// 解析当前目录下的文件/目录
	var children []*FileNode
	for _, line := range lines[3:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}

		child := &FileNode{
			Name: parts[3],
			Type: parts[0],
			Path: currentDir + "/" + parts[3],
		}

		if parts[0] == "F" {
			child.Size = utils.BytesToSize(parts[1])
			child.ModifiedTime = parts[2]
		} else {
			child.ModifiedTime = parts[2]
			child.Children = []*FileNode{}
		}

		children = append(children, child)
	}

	// 更新目录树
	updateFileTree(uid, currentDir, children)

	return UidFileBrowser[uid]
}

// 更新文件树的核心函数
func updateFileTree(uid, currentDir string, children []*FileNode) {
	// 将目录路径拆分为部分
	var parts []string
	if currentDir == "/" {
		parts = []string{}
	} else if strings.HasPrefix(currentDir, "/") {
		// Linux路径
		if currentDir != "/" {
			parts = strings.Split(currentDir[1:], "/")
		}
	} else if len(currentDir) >= 2 && currentDir[1] == ':' {
		// Windows路径
		if len(currentDir) > 2 {
			pathPart := currentDir[2:]
			if strings.HasPrefix(pathPart, "/") {
				pathPart = pathPart[1:]
			}
			if pathPart != "" {
				parts = strings.Split(pathPart, "/")
			}
		}
	}

	// 获取根节点
	root := getRootNode(uid, currentDir)
	if root == nil {
		return
	}

	// 导航到目标目录
	targetDir := root
	for _, part := range parts {
		found := false
		for _, child := range targetDir.Children {
			if child.Type == "D" && child.Name == part {
				targetDir = child
				found = true
				break
			}
		}
		if !found {
			// 创建不存在的目录
			var newPath string
			if targetDir.Path == "/" {
				newPath = "/" + part
			} else if strings.HasSuffix(targetDir.Path, "/") {
				newPath = targetDir.Path + part
			} else {
				newPath = targetDir.Path + "/" + part
			}

			newDir := &FileNode{
				Name:     part,
				Type:     "D",
				Path:     newPath,
				Children: []*FileNode{},
			}
			targetDir.Children = append(targetDir.Children, newDir)
			targetDir = newDir
		}
	}

	// 更新目标目录的子节点
	// 首先将现有节点转换为map以便快速查找
	existingMap := make(map[string]*FileNode)
	for _, child := range targetDir.Children {
		key := child.Name + ":" + child.Type
		existingMap[key] = child
	}

	// 准备新的子节点列表
	var newChildren []*FileNode

	// 添加或更新节点
	for _, newChild := range children {
		key := newChild.Name + ":" + newChild.Type
		if existingChild, exists := existingMap[key]; exists {
			// 更新现有节点
			existingChild.Size = newChild.Size
			existingChild.ModifiedTime = newChild.ModifiedTime
			// 对于目录，保留其原有子节点
			if newChild.Type == "D" {
				newChild.Children = existingChild.Children
			}
			newChildren = append(newChildren, existingChild)
			delete(existingMap, key)
		} else {
			// 添加新节点
			newChildren = append(newChildren, newChild)
		}
	}

	// 保留不在当前列表中的目录（但保留其子节点）
	for _, remaining := range existingMap {
		if remaining.Type == "D" {
			// 保留目录及其子节点
			newChildren = append(newChildren, remaining)
		}
		// 文件类型如果不在新列表中，将被删除
	}

	targetDir.Children = newChildren
}

// 获取根节点
func getRootNode(uid, path string) *FileNode {
	if _, exists := UidFileBrowser[uid]; !exists {
		return nil
	}

	// 确定根节点名称
	var rootName string
	if strings.HasPrefix(path, "/") {
		rootName = "/"
	} else if len(path) >= 2 && path[1] == ':' {
		rootName = path[:2]
	} else {
		return nil
	}

	for _, node := range UidFileBrowser[uid] {
		if node.Name == rootName {
			return node
		}
	}

	return nil
}

// 检查盘符是否存在
func exsitPan(filenode []*FileNode, pan string) bool {
	for _, file := range filenode {
		if file.Name == pan {
			return true
		}
	}
	return false
}

// 原来的辅助函数
func isInChild(root *FileNode, child *FileNode) bool {
	for _, childNode := range root.Children {
		if childNode.Name == child.Name && childNode.Type == child.Type {
			return true
		}
	}
	return false
}

func deleteChild(root []*FileNode, child *FileNode) []*FileNode {
	var result []*FileNode
	for _, childNode := range root {
		if childNode.Name != child.Name || childNode.Type != child.Type {
			result = append(result, childNode)
		}
	}
	return result
}
