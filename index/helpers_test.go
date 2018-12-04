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

	Convey("Test to split string in different languages", t, func() {
		So(splitStringToTerms("Привет, мир!"), ShouldResemble, []string{"Привет", "мир"})
		So(splitStringToTerms("Chào thế giới!"), ShouldResemble, []string{"Chào", "thế", "giới"})
		So(splitStringToTerms("ሰላም ልዑል!"), ShouldResemble, []string{"ሰላም", "ልዑል"})
		So(splitStringToTerms("Բարեւ աշխարհ!"), ShouldResemble, []string{"Բարեւ", "աշխարհ"})
		So(splitStringToTerms("ওহে বিশ্ব!"), ShouldResemble, []string{"ওহে", "বিশ্ব"})
		So(splitStringToTerms("你好 世界!"), ShouldResemble, []string{"你好", "世界"})
		So(splitStringToTerms("Γειά σου Κόσμε!"), ShouldResemble, []string{"Γειά", "σου", "Κόσμε"})
		So(splitStringToTerms("હેલો વર્લ્ડ!"), ShouldResemble, []string{"હેલો", "વર્લ્ડ"})
		So(splitStringToTerms("नमस्ते दुनिया!"), ShouldResemble, []string{"नमस्ते", "दुनिया"})
		So(splitStringToTerms("Გამარჯობა მსოფლიო!"), ShouldResemble, []string{"Გამარჯობა", "მსოფლიო"})
		So(splitStringToTerms("こんにちは世界!"), ShouldResemble, []string{"こんにちは世界"})
	})
}
