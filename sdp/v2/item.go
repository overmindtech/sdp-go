package sdp

func (i *Item) UniqueAttributeValue() string {
	return i.Attributes.GetFields()[i.UniqueAttribute].GetStringValue()
}

func (i *Item) ToReference() *Reference {
	return &Reference{
		Scope:                i.Scope,
		Type:                 i.Type,
		UniqueAttributeValue: i.UniqueAttributeValue(),
	}
}
