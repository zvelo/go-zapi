package zapitype

type ErrIncompleteResult QueryResult

func (e ErrIncompleteResult) Error() string {
	return "incomplete result"
}
