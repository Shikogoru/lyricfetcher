package console

import (
	"testing"
)

func TestConsoleOutput(t *testing.T) {
	// Ensure calls don't panic or error out on strings with CJK and Unicode
	FixEncoding()

	testStrings := []string{
		"Hello World\n",
		"晴空记事 - 小歌Sugar 清寒clair\n",
		"星与你消失之日【原唱重录版2022】【星と君が消えた日】\n",
		"Tiếng Việt có dấu: Cập nhật lời bài hát\n",
	}

	for _, s := range testStrings {
		n, err := WriteString(s)
		if err != nil {
			t.Errorf("WriteString(%q) returned error: %v", s, err)
		}
		if n <= 0 {
			t.Errorf("WriteString(%q) returned n = %d", s, n)
		}
	}

	// Test Print, Printf, Println
	if _, err := Print("test print: "); err != nil {
		t.Errorf("Print failed: %v", err)
	}
	if _, err := Printf("%s %d\n", "num", 42); err != nil {
		t.Errorf("Printf failed: %v", err)
	}
	if _, err := Println("test println"); err != nil {
		t.Errorf("Println failed: %v", err)
	}
}
