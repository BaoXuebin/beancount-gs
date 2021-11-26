package script

type AccountTypeSort []AccountType

func (s AccountTypeSort) Len() int {
	return len(s)
}

func (s AccountTypeSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s AccountTypeSort) Less(i, j int) bool {
	return s[i].Key <= s[j].Key
}

type AccountSort []Account

func (s AccountSort) Len() int {
	return len(s)
}

func (s AccountSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s AccountSort) Less(i, j int) bool {
	return s[i].Acc <= s[j].Acc
}
