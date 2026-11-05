package linear

const assignedIssuesQuery = `query AssignedIssues($first: Int!, $after: String, $filter: IssueFilter) {
	viewer {
		assignedIssues(first: $first, after: $after, filter: $filter, orderBy: updatedAt) {
			nodes {
				id
				identifier
				title
				url
				description
				createdAt
				updatedAt
				completedAt
				state { type }
				team { key }
				labels { nodes { name } }
				attachments { nodes { sourceType url } }
			}
			pageInfo { hasNextPage endCursor }
		}
	}
}`
