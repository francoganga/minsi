package field

type FieldDto struct {
	PropertyName string
	Value        any
	Label        string
	CssClass     string

	// TODO: Columns would be a string in the form of bootstrap columns
	Columns        string
	DefaultColumns string
	TemplateName   string

	// TODO: This could be usefull for defining the assets of every component
	//Assets []AssetsDto

}

// TODO: Then every specific field would contain the dto inside
type TextField struct {
	FieldDto
}

func NewTextField(propertyName string, label string) *TextField {

	dto := FieldDto{
		PropertyName:   propertyName,
		Label:          label,
		DefaultColumns: "col-md-6 col-xxl-5",
	}

	return &TextField{
		FieldDto: dto,
	}
}

