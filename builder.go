package graphql

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	// regular expression for the graphql variable name
	// reference: https://spec.graphql.org/June2018/#sec-Names
	regexVariableName = regexp.MustCompile(`\$([_A-Za-z][_0-9A-Za-z]*)`)

	errBuildQueryRequired = errors.New("no graphql query to be built")
)

type queryBuilderItem struct {
	query        string
	binding      interface{}
	requiredVars []string
}

// QueryBuilder is used to efficiently build dynamic queries and variables
// It helps construct multiple queries to a single request that needs to be conditionally added
type QueryBuilder struct {
	queries   []queryBuilderItem
	variables map[string]interface{}
}

// QueryBinding the type alias of interface tuple
// that includes the query string without fields and the binding type
type QueryBinding [2]interface{}

// NewQueryBuilder creates an empty QueryBuilder instance
func NewQueryBuilder() QueryBuilder {
	return QueryBuilder{
		variables: make(map[string]interface{}),
	}
}

// Query returns the new QueryBuilder with the inputted query
func (b QueryBuilder) Query(query string, binding interface{}) QueryBuilder {
	return QueryBuilder{
		queries: append(b.queries, queryBuilderItem{
			query,
			binding,
			findAllVariableNames(query),
		}),
		variables: b.variables,
	}
}

// Variables returns the new QueryBuilder with the inputted variables
func (b QueryBuilder) Variable(key string, value interface{}) QueryBuilder {
	return QueryBuilder{
		queries:   b.queries,
		variables: setMapValue(b.variables, key, value),
	}
}

// Variables returns the new QueryBuilder with the inputted variables
func (b QueryBuilder) Variables(variables map[string]interface{}) QueryBuilder {
	return QueryBuilder{
		queries:   b.queries,
		variables: mergeMap(b.variables, variables),
	}
}

// RemoveQuery returns the new QueryBuilder with query items and related variables removed
func (b QueryBuilder) Remove(query string, extra ...string) QueryBuilder {
	var newQueries []queryBuilderItem
	newVars := make(map[string]interface{})

	for _, q := range b.queries {
		if q.query == query || sliceStringContains(extra, q.query) {
			continue
		}
		newQueries = append(newQueries, q)
		if len(b.variables) > 0 {
			for _, k := range q.requiredVars {
				if v, ok := b.variables[k]; ok {
					newVars[k] = v
				}
			}
		}
	}

	return QueryBuilder{
		queries:   newQueries,
		variables: newVars,
	}
}

// RemoveQuery returns the new QueryBuilder with query items removed
// this method only remove query items only,
// to remove both query and variables, use Remove instead
func (b QueryBuilder) RemoveQuery(query string, extra ...string) QueryBuilder {
	var newQueries []queryBuilderItem

	for _, q := range b.queries {
		if q.query != query && !sliceStringContains(extra, q.query) {
			newQueries = append(newQueries, q)
		}
	}

	return QueryBuilder{
		queries:   newQueries,
		variables: b.variables,
	}
}

// RemoveQuery returns the new QueryBuilder with variable fields removed
func (b QueryBuilder) RemoveVariable(key string, extra ...string) QueryBuilder {
	newVars := make(map[string]interface{})
	for k, v := range b.variables {
		if k != key && !sliceStringContains(extra, k) {
			newVars[k] = v
		}
	}

	return QueryBuilder{
		queries:   b.queries,
		variables: newVars,
	}
}

// Build query and variable interfaces
func (b QueryBuilder) Build() ([]QueryBinding, map[string]interface{}, error) {
	if len(b.queries) == 0 {
		return nil, nil, errBuildQueryRequired
	}

	var requiredVars []string
	for _, q := range b.queries {
		requiredVars = append(requiredVars, q.requiredVars...)
	}
	variableLength := len(b.variables)
	requiredVariableLength := len(requiredVars)
	isMismatchedVariables := variableLength != requiredVariableLength
	if !isMismatchedVariables && requiredVariableLength > 0 {
		for _, varName := range requiredVars {
			if _, ok := b.variables[varName]; !ok {
				isMismatchedVariables = true
				break
			}
		}
	}
	if isMismatchedVariables {
		varNames := make([]string, 0, variableLength)
		for k := range b.variables {
			varNames = append(varNames, k)
		}
		return nil, nil, fmt.Errorf("mismatched variables; want: %+v; got: %+v", requiredVars, varNames)
	}

	query := make([]QueryBinding, 0, len(b.queries))
	for _, q := range b.queries {
		query = append(query, [2]interface{}{q.query, q.binding})
	}
	return query, b.variables, nil
}

func setMapValue(src map[string]interface{}, key string, value interface{}) map[string]interface{} {
	if src == nil {
		src = make(map[string]interface{})
	}
	src[key] = value
	return src
}

func mergeMap(src map[string]interface{}, dest map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range src {
		setMapValue(result, k, v)
	}
	for k, v := range dest {
		setMapValue(result, k, v)
	}
	return result
}

func sliceStringContains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func findAllVariableNames(query string) []string {
	var results []string
	for _, names := range regexVariableName.FindAllStringSubmatch(query, -1) {
		results = append(results, names[1])
	}
	return results
}
