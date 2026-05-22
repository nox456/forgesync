package notion

type DataSourceResponse struct {
	Object  string `json:"object"`
	Results []struct {
		Object     string `json:"object"`
		ID         string `json:"id"`
		Properties struct {
			Repo struct {
				RichText []struct {
					PlainText string `json:"plain_text"`
				} `json:"rich_text"`
			} `json:"repo"`
			Name struct {
				Title []struct {
					PlainText string `json:"plain_text"`
				} `json:"title"`
			} `json:"name"`
		} `json:"properties"`
	}
}
