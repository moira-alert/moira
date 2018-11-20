package index

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestStringsManipulations(t *testing.T) {
	Convey("Test escape symbols", t, func() {
		So(escapeString("12345"), ShouldResemble, "12345")
		So(escapeString("abcdefghijklmnop QRSTUVWXYZ"), ShouldResemble, "abcdefghijklmnop QRSTUVWXYZ")
		So(escapeString("I'm gonna use some.Bad.symbols.here. !!! Here we GO!!! (yeap, I mean it]"), ShouldResemble, "I m gonna use some Bad symbols here      Here we GO     yeap  I mean it ")
		So(escapeString(`+-=&|><!(){}[]^"'~*?\/`), ShouldResemble, "                      ")
	})

	Convey("Test split strings with symbols to escape", t, func() {
		So(splitStringToTerms("I.want.to.break:free!"), ShouldResemble, []string{"I", "want", "to", "break", "free"})
		So(splitStringToTerms("I;want-to,break_free!"), ShouldResemble, []string{"I", "want", "to", "break", "free"})
		So(splitStringToTerms(`I>want<to/break\free from&your@lies`), ShouldResemble, []string{"I", "want", "to", "break", "free", "from", "your", "lies"})
		So(splitStringToTerms(`(You)'[re] {so} "self" 'satisfied' |I| \don't/ ~need~ *you*`), ShouldResemble,
			[]string{"You", "re", "so", "self", "satisfied", "I", "don", "t", "need", "you"})
	})
}
