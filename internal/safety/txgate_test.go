package safety

import "testing"

func TestCheckWrongPhraseIsDryRun(t *testing.T) {
	dec := Check("nope")
	if !dec.DryRun || dec.Proceed {
		t.Errorf("Check(wrong phrase) = %+v, want DryRun=true Proceed=false", dec)
	}
}

func TestCheckEmptyPhraseIsDryRun(t *testing.T) {
	dec := Check("")
	if !dec.DryRun || dec.Proceed {
		t.Errorf("Check(\"\") = %+v, want DryRun=true Proceed=false", dec)
	}
}

func TestCheckCorrectPhraseProceeds(t *testing.T) {
	dec := Check(ConfirmPhrase)
	if dec.DryRun || !dec.Proceed {
		t.Errorf("Check(correct phrase) = %+v, want DryRun=false Proceed=true", dec)
	}
}
