package vgrouter

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"testing"
)

const childIdxThreshold = 5 // how many children causes use of index instead of list

type nd struct {

	// text bytes for this node
	text []byte

	// for a few children, a simple slice is efficient
	childList []nd

	// for a larger number of children, indexing by byte value is better
	childIdxStart int  // childIdx[0] corresponds to this byte value
	childIdx      []nd // slice of range of relevant children, childIdx[n-childIdxStart].text==nil means slot is unused

	// other data we want to store on the node, for use when we find a match, etc.
	item *item
}

// fixChildIdx will move childList to childIdx if len is >= childIdxThreshold
func (n *nd) fixChildIdx() {

	if len(n.childList) < childIdxThreshold {
		return
	}

	// first character of first child is index start value
	n.childIdxStart = int(n.childList[0].text[0])
	// figure out length first char of last child
	idxlen := int(n.childList[len(n.childList)-1].text[0]) - n.childIdxStart + 1

	n.childIdx = make([]nd, idxlen)

	// loop over the children and put each at the apppropriate position
	for _, childn := range n.childList {
		childni := int(childn.text[0]) - n.childIdxStart
		n.childIdx[childni] = childn
	}

	// childIdx now replaces childList
	n.childList = nil
}

// findExact will traverse the tree and find an exact match.
func (n *nd) findExact(s string) *item {

	thisn := *n

	ibase := 0
searching:
	for i := 0; i < len(s); i++ {

		// if we've run past the end of this node's text
		if i-ibase >= len(thisn.text) {

			// check childList
			for _, childn := range thisn.childList {
				if len(childn.text) == 0 {
					panic("childn.text is zero length, should not be possible")
				}
				if s[i] == childn.text[0] {
					ibase += len(thisn.text) // move ibase up
					thisn = childn           // this child becomes thisn
					continue searching
				}
			}

			// check childIdx
			if len(thisn.childIdx) > 0 {

				// calc offset into childIdx
				o := int(s[i]) - thisn.childIdxStart
				if o < len(thisn.childIdx) { // make sure it's not past the end
					nn := thisn.childIdx[o]
					if len(nn.text) > 0 { // make sure it's not an empty slot
						ibase += len(thisn.text) // move ibase up
						thisn = nn               // this child becomes thisn
						continue searching
					}
				}

			}

			// if no matching children, no match, we're done
			return nil

		} else if thisn.text[i-ibase] != s[i] { // if this node's text doesn't match, we're done
			return nil
		}
	}

	// if the loop above falls through, it's a match, return the item
	return thisn.item
}

func (n *nd) printTo(prefix, indent string, w io.Writer) error {

	cmode := "no_children"
	if len(n.childList) > 0 {
		cmode = "child_list"
	}
	if len(n.childIdx) > 0 {
		cmode = "child_index"
	}

	_, err := fmt.Fprintf(w, "%s%s (item=%#v; %s)\n", prefix, string(n.text), n.item, cmode)
	if err != nil {
		return err
	}

	// childList
	for _, nchild := range n.childList {
		err := nchild.printTo(prefix+strings.Repeat(" ", len(n.text))+indent, indent, w)
		if err != nil {
			return err
		}
	}

	emptyCount := 0
	printempty := func() error {
		if emptyCount > 0 {
			_, err := fmt.Fprintf(w, "%s-- %d empty index slot(s)\n", prefix+strings.Repeat(" ", len(n.text))+indent, emptyCount)
			emptyCount = 0
			return err
		}
		return nil
	}
	// childIdx
	for _, nchild := range n.childIdx {
		if len(nchild.text) == 0 {
			emptyCount++
			continue
		}
		err := printempty()
		if err != nil {
			return err
		}
		err = nchild.printTo(prefix+strings.Repeat(" ", len(n.text))+indent, indent, w)
		if err != nil {
			return err
		}
	}
	err = printempty()
	if err != nil {
		return err
	}

	return nil
}

// nextNd returns an nd that describes itemList[n].itemText[offset], with ndis being the subset of itemList
// that corresponds to the nd, and rest is the rest of the list.  itemList must be sorted ascending by itemText
func nextNd(offset int, itemList []*item) (n nd, ndis, rest []*item) {

	// skip past any at the beginning which are too short to have text at offset
	for len(itemList) > 0 && offset >= len(itemList[0].itemText) {
		itemList = itemList[1:] // trim first element
	}

	if len(itemList) == 0 {
		return
	}

	n.text = []byte{itemList[0].itemText[offset]}

	for i, item := range itemList {
		if item.itemText[offset] == n.text[0] {

			// if this item ends here
			if len(item.itemText) == offset+1 {
				// assign nd.item
				if n.item == nil {
					n.item = item
				} else { // but panic if already assigned (duplicate entry)
					panic(fmt.Errorf("two items have the same itemText %q: %#v; %#v", item.itemText, n.item, item))
				}
			}

			ndis = append(ndis, item)
		} else {
			rest = itemList[i:]
			return
		}
	}

	return
}

// index builds the tree from the given item ptr slice.
func index(offset int, itemList []*item) (ret []nd) {

	// step through itemList, one block of same itemText[offset] at a time
	for n, ndis, rest := nextNd(offset, itemList); n.text != nil; n, ndis, rest = nextNd(offset, rest) {
		childList := index(offset+1, ndis) // index children

		// if we should merge single child into n
		if len(childList) == 1 && n.item == nil {
			// save existing n.text
			tprefix := n.text
			// replace n with childList[0]
			n = childList[0]
			// add prefix in
			n.text = append(tprefix, n.text...)
		} else {
			n.childList = childList
			n.fixChildIdx()
		}

		ret = append(ret, n)
	}

	return
}

type item struct {
	itemText string
	itemData string
}

func TestTmp(t *testing.T) {

	itemList := []*item{
		&item{"/", "root"},
		&item{"/about", "about"},
		&item{"/app", "app"},
		&item{"/app/settings", "settings"},
		&item{"/app/setter", "setter"},
		// &item{"/app/setter", "setter2"},
		&item{"/app/setters", "setters"},
		&item{"/app/things", "things"},
		&item{"/all", "all"},
		&item{"/other", "other"},
		&item{"/another", "another"},
		&item{"/apoll", "a poll"},
		// dictionary snippet to test the childIdx stuff
		&item{"/dict/unneat", "unneat"},
		&item{"/dict/unneatly", "unneatly"},
		&item{"/dict/unneatness", "unneatness"},
		&item{"/dict/unnebulous", "unnebulous"},
		&item{"/dict/unnecessarily", "unnecessarily"},
		&item{"/dict/unnecessariness", "unnecessariness"},
		&item{"/dict/unnecessary", "unnecessary"},
		&item{"/dict/unnecessitated", "unnecessitated"},
		&item{"/dict/unnecessitating", "unnecessitating"},
		&item{"/dict/unnecessity", "unnecessity"},
		&item{"/dict/unneeded", "unneeded"},
		&item{"/dict/unneedful", "unneedful"},
		&item{"/dict/unneedfully", "unneedfully"},
		&item{"/dict/unneedfulness", "unneedfulness"},
		&item{"/dict/unneedy", "unneedy"},
		&item{"/dict/unnefarious", "unnefarious"},
		&item{"/dict/unnegated", "unnegated"},
		&item{"/dict/unneglected", "unneglected"},
		&item{"/dict/unnegligent", "unnegligent"},
		&item{"/dict/unnegotiable", "unnegotiable"},
		&item{"/dict/unnegotiableness", "unnegotiableness"},
		&item{"/dict/unnegotiably", "unnegotiably"},
		&item{"/dict/unnegotiated", "unnegotiated"},
		&item{"/dict/unneighbored", "unneighbored"},
		&item{"/dict/unneighborlike", "unneighborlike"},
		&item{"/dict/unneighborliness", "unneighborliness"},
		&item{"/dict/unneighborly", "unneighborly"},
		&item{"/dict/unnephritic", "unnephritic"},
		&item{"/dict/unnerve", "unnerve"},
		&item{"/dict/unnerved", "unnerved"},
		&item{"/dict/unnervous", "unnervous"},
		&item{"/dict/unnest", "unnest"},
		&item{"/dict/unnestle", "unnestle"},
		&item{"/dict/unnestled", "unnestled"},
		&item{"/dict/unneth", "unneth"},
		&item{"/dict/unnethe", "unnethe"},
		&item{"/dict/unnethes", "unnethes"},
		&item{"/dict/unnethis", "unnethis"},
		&item{"/dict/unnetted", "unnetted"},
		&item{"/dict/unnettled", "unnettled"},
		&item{"/dict/unneurotic", "unneurotic"},
		&item{"/dict/unneutral", "unneutral"},
		&item{"/dict/unneutralized", "unneutralized"},
		&item{"/dict/unneutrally", "unneutrally"},
	}

	sort.Slice(itemList, func(i, j int) bool {
		return itemList[i].itemText < itemList[j].itemText
	})

	nds := index(0, itemList)
	if len(nds) != 1 {
		panic("didn't get exactly one nd back from index")
	}
	n := nds[0]

	n.printTo("", "", os.Stdout)

	fmt.Println(`n.findExact("/app")`, n.findExact("/app"))
	fmt.Println(`n.findExact("/app")`, n.findExact("/app"))
	fmt.Println(`n.findExact("/app/settings")`, n.findExact("/app/settings"))
	fmt.Println(`n.findExact("/apoll")`, n.findExact("/apoll"))
	fmt.Println(`n.findExact("/notexist")`, n.findExact("/notexist"))
	fmt.Println(`n.findExact("/dict/unnestle")`, n.findExact("/dict/unnestle"))

}
