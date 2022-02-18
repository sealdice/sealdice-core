package main

func (self *Dice) RegisterBuiltinExt() {
	self.registerBuiltinExtCoc7()
	self.registerBuiltinExtLog()
	self.registerBuiltinExtFun()
}
