package telegram

import (
	"github.com/moira-alert/moira"
	"github.com/russross/blackfriday/v2"
	"regexp"
	"strings"
)

const (
	endSuffix = "...\n"
)

var (
	startHeaderRegexp = regexp.MustCompile("<h[0-9]+>")
	endHeaderRegexp   = regexp.MustCompile("</h[0-9]+>")
)

// descriptionFormatter is used for formatting trigger description for telegram.
// If maxSize == -1 then no limits will be applied to description,
// otherwise the description will be at most maxSize symbols (not bytes).
func descriptionFormatter(trigger moira.TriggerData, maxSize int) string {
	if maxSize == 0 {
		return ""
	}

	desc := trigger.Desc
	if trigger.Desc != "" {
		desc = trigger.Desc
		desc += "\n"
	}

	htmlDescStr := string(blackfriday.Run([]byte(desc)))

	// html headers are not supported by telegram html, so make them bold instead.
	htmlDescStr = startHeaderRegexp.ReplaceAllString(htmlDescStr, "<b>")
	replacedHeaders := endHeaderRegexp.ReplaceAllString(htmlDescStr, "</b>")

	// some tags are not supported or too long, so replace them.
	replacer := strings.NewReplacer(
		"<p>", "",
		"</p>", "",
		"&ldquo;", "&quot;",
		"&rdquo;", "&quot;",
		"<strong>", "<b>",
		"</strong>", "</b>",
		"<em>", "<i>",
		"</em>", "</i>",
		"<del>", "<s>",
		"</del>", "</s>")
	withReplacedTags := replacer.Replace(replacedHeaders)

	descRunes := []rune(withReplacedTags)
	if maxSize < 0 || len(descRunes) <= maxSize {
		return withReplacedTags
	}

	return cutDescription(descRunes, maxSize-len([]rune(endSuffix))) + endSuffix
}

type descriptionNodeType int8

const (
	undefined     descriptionNodeType = iota // if node type is undefined.
	openTag                                  // for example <b>.
	closeTag                                 // for example </b>.
	text                                     // text with no tags or escaped symbols.
	escapedSymbol                            // escaped symbols, for example '>' is turned into '&gt;'.
)

type descriptionNode struct {
	content []rune
	// start of content in the description
	start    int
	nodeType descriptionNodeType
}

// splitDescriptionIntoNodes converts html description into nodes. For example:
//
// "<b>Bold</b> smth &gt; smth"
//
// will be split to nodes with such content:
//
// ["<b>", "Bold", "</b>", " smth ", "&gt;", " smth"].
func splitDescriptionIntoNodes(fullDesc []rune, maxSize int) ([]descriptionNode, []int) {
	var nodes []descriptionNode
	var stack []int

	var nodeContent []rune
	prevNodeType := undefined
	startOfNode := 0
	for i := 0; i < maxSize; i++ {
		r := fullDesc[i]

		// tag started
		if r == '<' {
			if len(nodeContent) != 0 {
				nodes = append(nodes, descriptionNode{
					content:  nodeContent,
					start:    startOfNode,
					nodeType: prevNodeType,
				})
				nodeContent = []rune{}
			}
			prevNodeType = openTag
			startOfNode = i
			nodeContent = append(nodeContent, r)
			continue
		}

		if len(nodeContent) == 1 && nodeContent[0] == '<' && r == '/' {
			prevNodeType = closeTag
			nodeContent = append(nodeContent, r)
			continue
		}

		// start of escaped symbol
		if r == '&' {
			if len(nodeContent) != 0 {
				nodes = append(nodes, descriptionNode{
					content:  nodeContent,
					start:    startOfNode,
					nodeType: prevNodeType,
				})
				nodeContent = []rune{}
			}
			prevNodeType = escapedSymbol
			startOfNode = i
			nodeContent = append(nodeContent, r)
			continue
		}

		nodeContent = append(nodeContent, r)

		// tag ended
		if r == '>' {
			nodes = append(nodes, descriptionNode{
				content:  nodeContent,
				start:    startOfNode,
				nodeType: prevNodeType,
			})

			if prevNodeType == openTag {
				stack = append(stack, len(nodes)-1)
			} else if prevNodeType == closeTag {
				stack = stack[:len(stack)-1]
			}

			nodeContent = []rune{}
			prevNodeType = undefined
			startOfNode = i + 1
			continue
		}

		// end of escaped symbol
		if len(nodeContent) > 0 && nodeContent[0] == '&' && r == ';' {
			nodes = append(nodes, descriptionNode{
				content:  nodeContent,
				start:    startOfNode,
				nodeType: prevNodeType,
			})
			nodeContent = []rune{}
			prevNodeType = undefined
			startOfNode = i + 1
			continue
		}

		// not the tag nor escaped symbol
		if len(nodeContent) == 1 {
			prevNodeType = text
			startOfNode = i
		}
	}

	if len(nodeContent) != 0 && nodeContent[0] != '<' && nodeContent[0] != '&' {
		nodes = append(nodes, descriptionNode{
			content:  nodeContent,
			start:    startOfNode,
			nodeType: prevNodeType,
		})
	}

	return nodes, stack
}

// removeEmptyTags remove tag pairs like <b></b> from nodes.
func removeEmptyTags(nodes []descriptionNode) []descriptionNode {
	// After removing first empty pairs new may occur, so continue while no pairs are found.
	for {
		start := -1
		end := -1
		for i := 1; i < len(nodes); i++ {
			if nodes[i-1].nodeType == openTag && nodes[i].nodeType == closeTag {
				start = i - 1
				end = i
			}
		}

		if start == -1 && end == -1 {
			break
		}

		var newNodes []descriptionNode
		newNodes = append(newNodes, nodes[:start]...)
		if end <= len(nodes)-1 {
			newNodes = append(newNodes, nodes[end+1:]...)
		}

		nodes = newNodes
	}

	return nodes
}

// toString converts
func toString(nodes []descriptionNode) string {
	nodes = removeEmptyTags(nodes)

	var res string

	for _, node := range nodes {
		res += string(node.content)
	}

	return res
}

func lenContent(nodes []descriptionNode) int {
	res := 0

	for _, node := range nodes {
		res += len(node.content)
	}

	return res
}

func appendToHead(reversedCloseTags []descriptionNode, newNode descriptionNode) []descriptionNode {
	if len(reversedCloseTags) == 0 {
		reversedCloseTags = append(reversedCloseTags, newNode)
	} else {
		tail := reversedCloseTags
		reversedCloseTags = []descriptionNode{}
		reversedCloseTags = append(reversedCloseTags, newNode)
		reversedCloseTags = append(reversedCloseTags, tail...)
	}

	return reversedCloseTags
}

func cutDescription(fullDesc []rune, maxSize int) string {
	nodes, unclosed := splitDescriptionIntoNodes(fullDesc, maxSize)

	if len(unclosed) == 0 {
		return toString(nodes)
	}

	var reversedCloseTags []descriptionNode

	for i, nodeIdx := range unclosed {
		if string(nodes[nodeIdx].content) == "<pre>" {
			// remove <pre> block

			nodes = nodes[:nodeIdx]
			unclosed = unclosed[:i]
			break
		} else if strings.HasPrefix(string(nodes[nodeIdx].content), "<a href=\"") {
			nodes, reversedCloseTags, unclosed = cutLink(nodeIdx, nodes, reversedCloseTags, unclosed)
			break
		} else {
			// if we have such unclosed tags: <b>, <i>, <s> then
			// reversedCloseTags should be: </s>, </i>, </b>

			var newContent []rune
			newContent = append(newContent, nodes[nodeIdx].content[0], '/')
			newContent = append(newContent, nodes[nodeIdx].content[1:]...)

			reversedCloseTags = appendToHead(reversedCloseTags, descriptionNode{
				content:  newContent,
				nodeType: closeTag,
			})
		}
	}

	if len(unclosed) == 0 {
		return toString(nodes)
	}

	currentLen := lenContent(nodes)
	lenCloseTags := lenContent(reversedCloseTags)

	if maxSize < currentLen+lenCloseTags {
		for i := len(nodes) - 1; i >= 0; i-- {
			switch nodes[i].nodeType {
			case escapedSymbol:
				currentLen = lenContent(nodes[:i])
			case text:
				toCutLen := currentLen + lenCloseTags - maxSize

				// if we can cut text to give us space for
				if toCutLen >= len(nodes[i].content) {
					currentLen = lenContent(nodes[:i])
				} else {
					nodes[i].content = nodes[i].content[:len(nodes[i].content)-toCutLen]
					nodes = nodes[:i+1]
					goto cycleExited
				}
			case closeTag:
				reversedCloseTags = appendToHead(reversedCloseTags, descriptionNode{
					content:  nodes[i].content,
					nodeType: closeTag,
				})
				currentLen = lenContent(nodes[:i])
				lenCloseTags = lenContent(reversedCloseTags)
			case openTag:
				if len(reversedCloseTags) == 1 {
					nodes = nodes[:i]
					reversedCloseTags = []descriptionNode{}
					goto cycleExited
				}
				reversedCloseTags = reversedCloseTags[1:]
				lenCloseTags = lenContent(reversedCloseTags)
				currentLen = lenContent(nodes[:i])
			}

			if maxSize >= currentLen+lenCloseTags {
				nodes = nodes[:i]
				break
			}
		}
	}

cycleExited:
	nodes = append(nodes, reversedCloseTags...)

	return toString(nodes)
}

// cutLink returns new nodes, new reversedCloseTags and new unclosed. This function tries to cut link.
// By default, it will try to save link by removing tags from short link name and then cut `short link name`.
// If this two options doesn't work then it removes link.
//
// The `short link name` is like (on html example):
//
// <a href="link">short link name</a>.
func cutLink(nodeIdx int, nodes, reversedCloseTags []descriptionNode, unclosed []int) ([]descriptionNode, []descriptionNode, []int) {
	// try to save link, but if there is no space for closing all tags remove it and try to close other tags

	var noTagsNodes []descriptionNode
	skippedLen := 0
	textLen := 0

	for j := nodeIdx + 1; j < len(nodes); j++ {
		if nodes[j].nodeType == text || nodes[j].nodeType == escapedSymbol {
			if len(noTagsNodes) > 0 && nodes[j].nodeType == text && noTagsNodes[len(noTagsNodes)-1].nodeType == nodes[j].nodeType {
				// merge nodes with text types

				noTagsNodes[len(noTagsNodes)-1].content = append(noTagsNodes[len(noTagsNodes)-1].content, nodes[j].content...)
			} else {
				noTagsNodes = append(noTagsNodes, nodes[j])
			}

			textLen += len(nodes[j].content)
		} else {
			skippedLen += len(nodes[j].content)
		}
	}

	linkCloseRunes := []rune("</a>")
	lenReversedTags := lenContent(reversedCloseTags)
	if skippedLen+textLen > len(linkCloseRunes)+lenReversedTags {
		// we have enough space to close all tags and left the link

		if skippedLen < len(linkCloseRunes)+lenReversedTags {
			// we have enough space to close all tags, but we have to cut text inside of link

			newTextMaxLen := skippedLen + textLen - len(linkCloseRunes) - lenReversedTags

			// cutting text
			curLen := 0
			for noTagsNodesIdx := 0; noTagsNodesIdx < len(noTagsNodes); noTagsNodesIdx++ {
				if curLen+len(noTagsNodes[noTagsNodesIdx].content) < newTextMaxLen {
					// if text len + current len is lower than newTextMaxLen, then we use the whole content of the node and move to next

					curLen += len(noTagsNodes[noTagsNodesIdx].content)
				} else if curLen+len(noTagsNodes[noTagsNodesIdx].content) == newTextMaxLen {
					// if text len + current len is equal to newTextMaxLen, then we use the whole content and stop

					noTagsNodes = noTagsNodes[:noTagsNodesIdx+1]
					break
				} else {
					// with adding nodes[noTagsNodesIdx].content we overflow the newTextMaxLen, so we need to cut content of the node

					if noTagsNodes[noTagsNodesIdx].nodeType == escapedSymbol {
						// escaped symbols can not be cut, so stop

						noTagsNodes = noTagsNodes[:noTagsNodesIdx]
						break
					} else {
						// cut the content of the node and stop

						noTagsNodes[noTagsNodesIdx].content = noTagsNodes[noTagsNodesIdx].content[:newTextMaxLen-curLen]
						noTagsNodes = noTagsNodes[:noTagsNodesIdx+1]
						break
					}
				}
			}
		}

		// gather all nodes together
		nodes = append(nodes[:nodeIdx+1], noTagsNodes...)
		nodes = append(nodes, descriptionNode{
			content:  linkCloseRunes,
			nodeType: closeTag,
		})
		nodes = append(nodes, reversedCloseTags...)
		unclosed = []int{}
	} else {
		// there is no enough space for link, so remove it

		nodes = nodes[:nodeIdx]
		unclosed = unclosed[:len(reversedCloseTags)+1]
	}
	return nodes, reversedCloseTags, unclosed
}
