package notion

type NamedOption struct {
	Name string `json:"name"`
}

type RelationRef struct {
	ID string `json:"id"`
}

type TextContent struct {
	Content string `json:"content"`
}

type RichTextItem struct {
	PlainText string `json:"plain_text"`
}

type DateValue struct {
	Start string `json:"start"`
}

type RichTextInput struct {
	Text TextContent `json:"text"`
}

type NumberProperty struct {
	Number int `json:"number"`
}

type URLProperty struct {
	URL string `json:"url"`
}

type StatusProperty struct {
	Status NamedOption `json:"status"`
}

type SelectProperty struct {
	Select NamedOption `json:"select"`
}

type MultiSelectProperty struct {
	MultiSelect []NamedOption `json:"multi_select"`
}

type RelationProperty struct {
	Relation []RelationRef `json:"relation"`
}

type DateProperty struct {
	Date DateValue `json:"date"`
}

type CreatedTimeProperty struct {
	Date string `json:"created_time"`
}

type TitleProperty struct {
	Title []RichTextItem `json:"title"`
}

type RichTextProperty struct {
	RichText []RichTextItem `json:"rich_text"`
}

type TitleInputProperty struct {
	Title []RichTextInput `json:"title"`
}

type PageParent struct {
	DataSourceID string `json:"data_source_id"`
}

type ReplaceContent struct {
	NewStr string `json:"new_str"`
}

type ProjectsDataSourceResponse struct {
	Object  string `json:"object"`
	Results []struct {
		Object     string `json:"object"`
		ID         string `json:"id"`
		Properties struct {
			Repo RichTextProperty `json:"repo"`
			Name TitleProperty    `json:"name"`
		} `json:"properties"`
	}
}

type StoriesDataSourceResponse struct {
	Object  string `json:"object"`
	Results []struct {
		Object     string `json:"object"`
		ID         string `json:"id"`
		Properties struct {
			Issue        NumberProperty      `json:"Issue"`
			CreatedTime  CreatedTimeProperty `json:"Created time"`
			Labels       MultiSelectProperty `json:"Labels"`
			LastWorkedAt DateProperty        `json:"Last Worked At"`
			FinishedDate DateProperty        `json:"Finished Date"`
			Status       StatusProperty      `json:"Status"`
			Project      RelationProperty    `json:"Project"`
			URL          URLProperty         `json:"URL"`
			Name         TitleProperty       `json:"Name"`
		} `json:"properties"`
	}
}

type StoryMarkdownResponse struct {
	Object   string `json:"object"`
	Markdown string `json:"markdown"`
}

type NumberFilter struct {
	Equals int `json:"equals"`
}

type RelationFilter struct {
	Contains string `json:"contains"`
}

type PropertyFilter struct {
	Property string          `json:"property"`
	Number   *NumberFilter   `json:"number,omitempty"`
	Relation *RelationFilter `json:"relation,omitempty"`
}

type FilterCondition struct {
	And []PropertyFilter `json:"and,omitempty"`
	Or  []PropertyFilter `json:"or,omitempty"`
}

type StoryFilterPayload struct {
	Filter FilterCondition `json:"filter"`
}

type StoryProperties struct {
	Name         *TitleInputProperty  `json:"Name,omitempty"`
	Project      *RelationProperty    `json:"Project,omitempty"`
	Issue        *NumberProperty      `json:"Issue,omitempty"`
	URL          *URLProperty         `json:"URL,omitempty"`
	Status       *StatusProperty      `json:"Status,omitempty"`
	Labels       *MultiSelectProperty `json:"Labels,omitempty"`
	LastWorkedAt *DateProperty        `json:"Last Worked At,omitempty"`
	FinishedDate *DateProperty        `json:"Finished Date,omitempty"`
}

type StoryCreatePayload struct {
	Parent     PageParent      `json:"parent"`
	Properties StoryProperties `json:"properties"`
	Markdown   string          `json:"markdown"`
}

type StoryUpdatePayload struct {
	Properties StoryProperties `json:"properties"`
}

type StoryMarkdownUpdatePayload struct {
	Type           string         `json:"type"`
	ReplaceContent ReplaceContent `json:"replace_content"`
}
