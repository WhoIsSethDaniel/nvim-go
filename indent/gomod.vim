if exists("b:did_indent")
  finish
endif
let b:did_indent = 1

setlocal autoindent
setlocal indentkeys+=<:>,0=},0=)
setlocal noexpandtab
setlocal nolisp
