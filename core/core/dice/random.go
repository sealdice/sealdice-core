package dice

func random() {

}

var store map[string]string = make(map[string]string);

func CmdRegister(name string, args string) {
	store[name] = args;
}

func CmdList() []string {
	lst := make([]string, 0, 5);

	for name := range store {
		lst = append(lst, name);
	}

	return lst;
}
