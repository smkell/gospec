// Copyright © 2009-2010 Esko Luontola <www.orfjackal.net>
// This software is released under the Apache License 2.0.
// The license text is at http://www.apache.org/licenses/LICENSE-2.0

package gospec

import (
	"bytes"
	"nanospec"
	"os"
	"strings"
)


func ResultsSpec(c nanospec.Context) {
	results := newResultCollector()

	c.Specify("When results have many root specs", func() {
		results.Update(newSpecRun("RootSpec2", nil, nil, nil))
		results.Update(newSpecRun("RootSpec1", nil, nil, nil))
		results.Update(newSpecRun("RootSpec3", nil, nil, nil))

		c.Specify("then the roots are sorted alphabetically", func() {
			c.Expect(results).Matches(ReportIs(`
- RootSpec1
- RootSpec2
- RootSpec3

3 specs, 0 failures
`))
		})
	})

	c.Specify("When results have many child specs", func() {
		// In tests, when a spec has many children, make sure
		// to pass a common parent instance to all the siblings.
		// Otherwise the parent's numberOfChildren is not
		// incremented and the children's paths will be wrong.

		// use names which would not sort alphabetically
		root := newSpecRun("RootSpec", nil, nil, nil)
		child1 := newSpecRun("one", nil, root, nil)
		child2 := newSpecRun("two", nil, root, nil)
		child3 := newSpecRun("three", nil, root, nil)

		// register in random order
		results.Update(root)
		results.Update(child1)

		results.Update(root)
		results.Update(child3)

		results.Update(root)
		results.Update(child2)

		c.Specify("then the children are sorted by their declaration order", func() {
			c.Expect(results).Matches(ReportIs(`
- RootSpec
  - one
  - two
  - three

4 specs, 0 failures
`))
		})
	})

	c.Specify("Case: zero specs", func() {
		c.Expect(results).Matches(ReportIs(`
0 specs, 0 failures
`))
	})
	c.Specify("Case: spec with no children", func() {
		a1 := newSpecRun("RootSpec", nil, nil, nil)
		results.Update(a1)
		c.Expect(results).Matches(ReportIs(`
- RootSpec

1 specs, 0 failures
`))
	})
	c.Specify("Case: spec with a child", func() {
		a1 := newSpecRun("RootSpec", nil, nil, nil)
		a2 := newSpecRun("Child A", nil, a1, nil)
		results.Update(a1)
		results.Update(a2)
		c.Expect(results).Matches(ReportIs(`
- RootSpec
  - Child A

2 specs, 0 failures
`))
	})
	c.Specify("Case: spec with nested children", func() {
		a1 := newSpecRun("RootSpec", nil, nil, nil)
		a2 := newSpecRun("Child A", nil, a1, nil)
		a3 := newSpecRun("Child AA", nil, a2, nil)
		results.Update(a1)
		results.Update(a2)
		results.Update(a3)
		c.Expect(results).Matches(ReportIs(`
- RootSpec
  - Child A
    - Child AA

3 specs, 0 failures
`))
	})
	c.Specify("Case: spec with multiple nested children", func() {
		runner := NewRunner()
		runner.AddSpec("DummySpecWithMultipleNestedChildren", DummySpecWithMultipleNestedChildren)
		runner.Run()
		c.Expect(runner.Results()).Matches(ReportIs(`
- DummySpecWithMultipleNestedChildren
  - Child A
    - Child AA
    - Child AB
  - Child B
    - Child BA
    - Child BB
    - Child BC

8 specs, 0 failures
`))
	})

	c.Specify("When specs fail", func() {
		a1 := newSpecRun("Failing", nil, nil, nil)
		a1.AddError(newError("X did not equal Y", currentLocation()))
		results.Update(a1)

		b1 := newSpecRun("Passing", nil, nil, nil)
		b2 := newSpecRun("Child failing", nil, b1, nil)
		b2.AddError(newError("moon was not cheese", currentLocation()))
		results.Update(b1)
		results.Update(b2)

		c.Specify("then the errors are reported", func() {
			c.Expect(results).Matches(ReportIs(`
- Failing [FAIL]
    X did not equal Y
- Passing
  - Child failing [FAIL]
      moon was not cheese

3 specs, 2 failures
`))
		})
	})
	c.Specify("When spec passes on 1st run but fails on 2nd run", func() {
		i := 0
		runner := NewRunner()
		runner.AddSpec("RootSpec", func(c Context) {
			if i == 1 {
				c.Then(10).Should.Equal(20)
			}
			i++
			c.Specify("Child A", func() {})
			c.Specify("Child B", func() {})
		})
		runner.Run()

		c.Specify("then the error is reported", func() {
			c.Expect(runner.Results()).Matches(ReportIs(`
- RootSpec [FAIL]
    Expected '20' but was '10'
  - Child A
  - Child B

3 specs, 1 failures
`))
		})
	})
	c.Specify("When root spec fails sporadically", func() {
		runner := NewRunner()
		runner.AddSpec("RootSpec", func(c Context) {
			i := 0
			c.Specify("Child A", func() {
				i = 1
			})
			c.Specify("Child B", func() {
				i = 2
			})
			c.Then(10).Should.Equal(20)     // stays same - will be reported once
			c.Then(10 + i).Should.Equal(20) // changes - will be reported many times
		})
		runner.Run()

		c.Specify("then the errors are merged together", func() {
			c.Expect(runner.Results()).Matches(ReportIs(`
- RootSpec [FAIL]
    Expected '20' but was '10'
    Expected '20' but was '11'
    Expected '20' but was '12'
  - Child A
  - Child B

3 specs, 1 failures
`))
		})
	})
	c.Specify("When non-root spec fails sporadically", func() {
		runner := NewRunner()
		runner.AddSpec("RootSpec", func(c Context) {
			c.Specify("Failing", func() {
				i := 0
				c.Specify("Child A", func() {
					i = 1
				})
				c.Specify("Child B", func() {
					i = 2
				})
				c.Then(10).Should.Equal(20)     // stays same - will be reported once
				c.Then(10 + i).Should.Equal(20) // changes - will be reported many times
			})
		})
		runner.Run()

		c.Specify("then the errors are merged together", func() {
			c.Expect(runner.Results()).Matches(ReportIs(`
- RootSpec
  - Failing [FAIL]
      Expected '20' but was '10'
      Expected '20' but was '11'
      Expected '20' but was '12'
    - Child A
    - Child B

4 specs, 1 failures
`))
		})
	})
}

func ReportIs(expected string) nanospec.Matcher {
	return func(v interface{}) os.Error {
		out := new(bytes.Buffer)
		results := v.(*ResultCollector)
		results.Visit(NewPrinter(SimplePrintFormat(out)))

		actual := strings.TrimSpace(out.String())
		expected = strings.TrimSpace(expected)
		if actual != expected {
			return os.ErrorString("Expected report:\n" + expected + "\n\nBut was:\n" + actual)
		}
		return nil
	}
}
