package go_cypherdsl

import (
	"errors"
	"fmt"
	"strings"
)

type ConditionBuilder struct {
	Start   *operatorNode
	Current *conditionNode
	errors  []error
}

// C is used to start a condition chain provided with a ConditionConfig
func C(condition *ConditionConfig) ConditionOperator {
	wq, err := NewCondition(condition)

	if err != nil {
		return &ConditionBuilder{
			errors: []error{err},
		}
	}

	cond := &ConditionBuilder{
		Start:   nil,
		errors:  nil,
		Current: nil,
	}

	node := &operatorNode{
		First: true,
		Query: &conditionNode{
			Condition: wq,
			Next:      nil,
		},
	}

	err = cond.addNext(node)
	if err != nil {
		cond.addError(err)
	}

	return cond
}

//add an error to the condition chain
func (c *ConditionBuilder) addError(e error) {
	if c.errors == nil {
		c.errors = []error{e}
	} else {
		c.errors = append(c.errors, e)
	}
}

//check if the builder has had any errors down the chain
func (c *ConditionBuilder) hasErrors() bool {
	return c.errors != nil && len(c.errors) > 0
}

//add another node to the chain
func (c *ConditionBuilder) addNext(node *operatorNode) error {
	if node == nil {
		return errors.New("node can not be nil")
	}

	if node.Query == nil {
		return errors.New("next can not be nil")
	}

	//different behavior if it's the first of the chain
	if c.Start == nil {
		c.Start = node
		c.Current = node.Query
	} else {
		c.Current.Next = node
		c.Current = node.Query
	}

	return nil
}

func (c *ConditionBuilder) And(condition *ConditionConfig) ConditionOperator {
	return c.addCondition(condition, "AND")
}

func (c *ConditionBuilder) Or(condition *ConditionConfig) ConditionOperator {
	return c.addCondition(condition, "OR")
}

func (c *ConditionBuilder) Xor(condition *ConditionConfig) ConditionOperator {
	return c.addCondition(condition, "XOR")
}

func (c *ConditionBuilder) Not(condition *ConditionConfig) ConditionOperator {
	return c.addCondition(condition, "NOT")
}

func (c *ConditionBuilder) AndNot(condition *ConditionConfig) ConditionOperator {
	return c.addCondition(condition, "AND NOT")
}

func (c *ConditionBuilder) AndNested(query WhereQuery, err error) ConditionOperator {
	return c.addNestedCondition(query, err, "AND")
}

func (c *ConditionBuilder) OrNested(query WhereQuery, err error) ConditionOperator {
	return c.addNestedCondition(query, err, "OR")
}

func (c *ConditionBuilder) XorNested(query WhereQuery, err error) ConditionOperator {
	return c.addNestedCondition(query, err, "XOR")
}

func (c *ConditionBuilder) NotNested(query WhereQuery, err error) ConditionOperator {
	return c.addNestedCondition(query, err, "NOT")
}

func (c *ConditionBuilder) AndNotNested(query WhereQuery, err error) ConditionOperator {
	return c.addNestedCondition(query, err, "AND NOT")
}

func (c *ConditionBuilder) addNestedCondition(query WhereQuery, err error, condType string) ConditionOperator {
	if c.hasErrors() {
		return c
	}

	if err != nil {
		c.addError(err)
		return c
	}

	//create node, make sure to wrap the query in parentheses since it's nested.
	node := &operatorNode{
		Condition: condType,
		Query: &conditionNode{
			Condition: "(" + query + ")",
			Next:      nil,
		},
	}

	//add it
	err = c.addNext(node)
	if err != nil {
		c.addError(err)
	}

	//return pointer to the builder
	return c
}

func (c *ConditionBuilder) addCondition(condition *ConditionConfig, condType string) ConditionOperator {
	//check if any errors are present, if they are, bail
	if c.hasErrors() {
		return c
	}

	//convert condition object into actual cypher
	wq, err := NewCondition(condition)
	if err != nil {
		c.addError(err)
		return c
	}

	//create the next node of the linked list
	node := &operatorNode{
		Condition: condType,
		Query: &conditionNode{
			Condition: wq,
			Next:      nil,
		},
	}

	//add it
	err = c.addNext(node)
	if err != nil {
		c.addError(err)
	}

	//return pointer to the builder
	return c
}

func (c *ConditionBuilder) Build() (WhereQuery, error) {
	//if it has errors, compile that and return
	if c.hasErrors() {
		errStr := ""
		for _, err := range c.errors {
			errStr += err.Error() + ";"
		}

		errStr = strings.TrimSuffix(errStr, ";")

		return "", fmt.Errorf("(%v) errors occurred: %s", len(c.errors), errStr)
	}

	query := ""

	//if start is not defined, something went wrong
	if c.Start == nil {
		return "", errors.New("no condition defined")
	}

	i := c.Start

	//iterate...
	for {
		if i == nil || i.Query == nil {
			break
		}

		t := ""

		if i.First {

		} else {
			t += i.Condition + " "
		}

		query += t + i.Query.Condition.ToString() + " "

		//iterate up
		i = i.Query.Next
	}

	//return entire condition
	return WhereQuery(strings.TrimSuffix(query, " ")), nil
}

type conditionNode struct {
	Condition WhereQuery
	Next      *operatorNode
}

type operatorNode struct {
	First     bool
	Condition string
	Query     *conditionNode
}

type ConditionOperator interface {
	And(c *ConditionConfig) ConditionOperator
	AndNested(query WhereQuery, err error) ConditionOperator
	Or(c *ConditionConfig) ConditionOperator
	OrNested(query WhereQuery, err error) ConditionOperator
	Xor(c *ConditionConfig) ConditionOperator
	XorNested(query WhereQuery, err error) ConditionOperator
	Not(c *ConditionConfig) ConditionOperator
	NotNested(query WhereQuery, err error) ConditionOperator
	AndNot(c *ConditionConfig) ConditionOperator
	AndNotNested(query WhereQuery, err error) ConditionOperator
	Build() (WhereQuery, error)
}

type BooleanOperator string

const (
	LessThanOperator             BooleanOperator = "<"
	GreaterThanOperator          BooleanOperator = ">"
	LessThanOrEqualToOperator    BooleanOperator = "<="
	GreaterThanOrEqualToOperator BooleanOperator = ">="
	EqualToOperator              BooleanOperator = "="
	InOperator                   BooleanOperator = "IN"
	IsOperator                   BooleanOperator = "IS"
	RegexEqualToOperator         BooleanOperator = "=~"
	StartsWithOperator           BooleanOperator = "STARTS WITH"
	EndsWithOperator             BooleanOperator = "ENDS WITH"
	ContainsOperator             BooleanOperator = "CONTAINS"
)

// ConditionConfig is the configuration object for where conditions
type ConditionConfig struct {
	// Operators that can be used
	ConditionOperator BooleanOperator

	// Condition functions that can be used
	ConditionFunction string

	Name  string
	Field string
	Label string

	//exclude parentheses
	FieldManipulationFunction string

	// When using any operator to compare to a specific value (other than InOperator), this field must be specified.
	// If Check and (CheckName, CheckField) are both specified - Check will take precedence.
	Check interface{}

	// When comparing one node to another, CheckField and CheckName must be specified. CheckLabel is optional
	CheckName  string
	CheckField string
	CheckLabel string

	// When using the InOperator, this field must be specified
	CheckSlice []interface{}

	// When NegateCondition is set to true, NOT is appended to the start of this condition
	NegateCondition bool
}

func (condition *ConditionConfig) ToString() (string, error) {
	//check initial error conditions
	if condition.Name == "" && condition.Field == "" && condition.Label == "" && condition.FieldManipulationFunction == "" {
		return "", errors.New("all of Name, Field, Function, and Label must be specified")
	}

	if condition.Field != "" && condition.Label != "" && condition.FieldManipulationFunction != "" {
		return "", errors.New("not all of Field, Label, and FieldManipulationFunction can be specified")
	}

	query := ""

	if condition.NegateCondition {
		query += "NOT "
	}

	// Build the fields
	if condition.Field != "" {
		query += fmt.Sprintf("%s.%s", condition.Name, condition.Field)
	} else if condition.Label != "" {
		//we're done here
		return fmt.Sprintf("%s:%s", condition.Name, condition.Label), nil
	} else {
		query += condition.Name
	}

	if condition.FieldManipulationFunction != "" {
		query += fmt.Sprintf("%s(%s)", condition.FieldManipulationFunction, query)
	}

	if condition.ConditionOperator == "" && condition.ConditionFunction == "" {
		return "", errors.New("one of ConditionOperator or ConditionFunction must be specified")
	}

	if condition.ConditionOperator != "" && condition.ConditionFunction != "" {
		return "", errors.New("only one of ConditionOperator or ConditionFunction can be specified")
	}

	// build the operators
	if condition.ConditionOperator != "" {
		query += fmt.Sprintf(" %s", condition.ConditionOperator)
	} else if condition.ConditionFunction != "" {
		if condition.NegateCondition {
			return fmt.Sprintf("NOT %s(%s)", condition.ConditionFunction, strings.Trim(query, "NOT ")), nil
		}
		return fmt.Sprintf("%s(%s)", condition.ConditionFunction, query), nil
	}

	//check if it's valid for in
	if condition.ConditionOperator == InOperator {
		if condition.CheckSlice == nil {
			return "", errors.New("slice can not be nil")
		}

		if condition.Check != nil {
			return "", errors.New("check should not be defined when using in operator")
		}

		if len(condition.CheckSlice) == 0 {
			return "", errors.New("slice should not be nil")
		}

		q := "["

		for _, val := range condition.CheckSlice {
			str, err := cypherizeInterface(val)
			if err != nil {
				return "", err
			}

			q += fmt.Sprintf("%s,", str)
		}

		query += " " + strings.TrimSuffix(q, ",") + "]"
	} else {
		if condition.Check != nil {
			str, err := cypherizeInterface(condition.Check)
			if err != nil {
				return "", err
			}
			query += " " + str
		} else if condition.CheckName != "" && condition.CheckField != "" {
			query += fmt.Sprintf(" %s.%s", condition.CheckName, condition.CheckField)
		} else {
			return "", errors.New("one of (Check) or (CheckName, CheckField) must be specified")
		}
	}

	return query, nil
}

func NewCondition(condition *ConditionConfig) (WhereQuery, error) {
	if condition == nil {
		return "", errors.New("condition can not be nil")
	}

	str, err := condition.ToString()
	if err != nil {
		return "", err
	}

	return WhereQuery(str), nil
}
