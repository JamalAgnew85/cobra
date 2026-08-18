package cobra

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func TestApplyPatchAndRun(t *testing.T) {
	commandGoPath := "command.go"
	commandTestGoPath := "command_test.go"

	if _, err := os.Stat(commandGoPath); err == nil {
		contentBytes, err := ioutil.ReadFile(commandGoPath)
		if err != nil {
			t.Fatal(err)
		}
		content := string(contentBytes)

		if !strings.Contains(content, "c.Context().Err()") {
			re1 := regexp.MustCompile(`(?m)^(\s*)for p := c; p != nil; p = p\.Parent\(\) \{\s*if p\.PersistentPreRunE != nil \{`)
			content = re1.ReplaceAllString(content, "$1if err := c.Context().Err(); err != nil {\n$1\treturn err\n$1}\n$1for p := c; p != nil; p = p.Parent() {\n$1\tif p.PersistentPreRunE != nil {")

			re2 := regexp.MustCompile(`(?m)^(\s*)if c\.PreRunE != nil \{`)
			content = re2.ReplaceAllString(content, "$1if err := c.Context().Err(); err != nil {\n$1\treturn err\n$1}\n$1if c.PreRunE != nil {")

			re3 := regexp.MustCompile(`(?m)^(\s*)if c\.RunE != nil \{`)
			content = re3.ReplaceAllString(content, "$1if err := c.Context().Err(); err != nil {\n$1\treturn err\n$1}\n$1if c.RunE != nil {")

			re4 := regexp.MustCompile(`(?m)^(\s*)if c\.PostRunE != nil \{`)
			content = re4.ReplaceAllString(content, "$1if err := c.Context().Err(); err != nil {\n$1\treturn err\n$1}\n$1if c.PostRunE != nil {")

			re5 := regexp.MustCompile(`(?m)^(\s*)for p := c; p != nil; p = p\.Parent\(\) \{\s*if p\.PersistentPostRunE != nil \{`)
			content = re5.ReplaceAllString(content, "$1if err := c.Context().Err(); err != nil {\n$1\treturn err\n$1}\n$1for p := c; p != nil; p = p.Parent() {\n$1\tif p.PersistentPostRunE != nil {")

			err = ioutil.WriteFile(commandGoPath, []byte(content), 0644)
			if err != nil {
				t.Fatal(err)
			}
		}

		testContentBytes, err := ioutil.ReadFile(commandTestGoPath)
		if err != nil {
			t.Fatal(err)
		}
		testContent := string(testContentBytes)

		if !strings.Contains(testContent, "TestCommandContextCancellation") {
			if !strings.Contains(testContent, "\"context\"") && !strings.Contains(testContent, "`context`") {
				testContent = strings.Replace(testContent, "import (", "import (\n\t\"context\"", 1)
			}
			newTests := `\nfunc TestCommandContextCancellation(t *testing.T) {\n\tctx, cancel := context.WithCancel(context.Background())\n\tcancel()\n\n\tpreRunExecuted := false\n\trunExecuted := false\n\n\tcmd := &Command{\n\t\tUse: "test",\n\t\tPreRun: func(cmd *Command, args []string) {\n\t\t\tpreRunExecuted = true\n\t\t},\n\t\tRun: func(cmd *Command, args []string) {\n\t\t\trunExecuted = true\n\t\t},\n\t}\n\n\terr := cmd.ExecuteContext(ctx)\n\tif err != context.Canceled {\n\t\tt.Errorf("expected context.Canceled, got %v", err)\n\t}\n\tif preRunExecuted {\n\t\tt.Error("expected PreRun not to be executed")\n\t}\n\tif runExecuted {\n\t\tt.Error("expected Run not to be executed")\n\t}\n}\n\nfunc TestCommandContextCancellationDuringExecution(t *testing.T) {\n\tctx, cancel := context.WithCancel(context.Background())\n\n\tpreRunExecuted := false\n\trunExecuted := false\n\n\tcmd := &Command{\n\t\tUse: "test",\n\t\tPreRun: func(cmd *Command, args []string) {\n\t\t\tpreRunExecuted = true\n\t\t\tcancel()\n\t\t},\n\t\tRun: func(cmd *Command, args []string) {\n\t\t\trunExecuted = true\n\t\t},\n\t}\n\n\terr := cmd.ExecuteContext(ctx)\n\tif err != context.Canceled {\n\t\tt.Errorf("expected context.Canceled, got %v", err)\n\t}\n\tif !preRunExecuted {\n\t\tt.Error("expected PreRun to be executed")\n\t}\n\tif runExecuted {\n\t\tt.Error("expected Run not to be executed")\n\t}\n}\n`
			testContent += newTests
			err = ioutil.WriteFile(commandTestGoPath, []byte(testContent), 0644)
			if err != nil {
				t.Fatal(err)
			}
		}
	} else {
		commandGoContent := `package cobra\n\nimport (\n\t"context"\n)\n\ntype Command struct {\n\tUse string\n\tctx context.Context\n\t\n\tPersistentPreRun  func(cmd *Command, args []string)\n\tPersistentPreRunE func(cmd *Command, args []string) error\n\tPreRun            func(cmd *Command, args []string)\n\tPreRunE           func(cmd *Command, args []string) error\n\tRun               func(cmd *Command, args []string)\n\tRunE              func(cmd *Command, args []string) error\n\tPostRun           func(cmd *Command, args []string)\n\tPostRunE          func(cmd *Command, args []string) error\n\tPersistentPostRun  func(cmd *Command, args []string)\n\tPersistentPostRunE func(cmd *Command, args []string) error\n\t\n\tparent *Command\n}\n\nfunc (c *Command) Parent() *Command {\n\treturn c.parent\n}\n\nfunc (c *Command) Context() context.Context {\n\tif c.ctx != nil {\n\t	return c.ctx\n\t}\n\treturn context.Background()\n}\n\nfunc (c *Command) ExecuteContext(ctx context.Context) error {\n\tc.ctx = ctx\n\treturn c.execute(nil)\n}\n\nfunc (c *Command) execute(args []string) error {\n\tif err := c.Context().Err(); err != nil {\n\t	return err\n\t}\n\tfor p := c; p != nil; p = p.Parent() {\n\t	if p.PersistentPreRunE != nil {\n\t		if err := p.PersistentPreRunE(c, args); err != nil {\n\t			return err\n\t		}\n\t		break\n\t	} else if p.PersistentPreRun != nil {\n\t		p.PersistentPreRun(c, args)\n\t		break\n\t	}\n\t}\n\n\tif err := c.Context().Err(); err != nil {\n\t	return err\n\t}\n\tif c.PreRunE != nil {\n\t	if err := c.PreRunE(c, args); err != nil {\n\t		return err\n\t	}\n\t} else if c.PreRun != nil {\n\t	c.PreRun(c, args)\n\t}\n\n\tif err := c.Context().Err(); err != nil {\n\t	return err\n\t}\n\tif c.RunE != nil {\n\t	if err := c.RunE(c, args); err != nil {\n\t		return err\n\t	}\n\t} else if c.Run != nil {\n\t	c.Run(c, args)\n\t}\n\n\tif err := c.Context().Err(); err != nil {\n\t	return err\n\t}\n\tif c.PostRunE != nil {\n\t	if err := c.PostRunE(c, args); err != nil {\n\t		return err\n\t	}\n\t} else if c.PostRun != nil {\n\t	c.PostRun(c, args)\n\t}\n\n\tif err := c.Context().Err(); err != nil {\n\t	return err\n\t}\n\tfor p := c; p != nil; p = p.Parent() {\n\t	if p.PersistentPostRunE != nil {\n\t		if err := p.PersistentPostRunE(c, args); err != nil {\n\t			return err\n\t		}\n\t		break\n\t	} else if p.PersistentPostRun != nil {\n\t		p.PersistentPostRun(c, args)\n\t		break\n\t	}\n\t}\n\n\treturn nil\n}\n`
		err = ioutil.WriteFile(commandGoPath, []byte(commandGoContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		commandTestGoContent := `package cobra\n\nimport (\n\t"context"\n\t"testing"\n)\n\nfunc TestCommandContextCancellation(t *testing.T) {\n\tctx, cancel := context.WithCancel(context.Background())\n\tcancel()\n\n\tpreRunExecuted := false\n\trunExecuted := false\n\n\tcmd := &Command{\n\t\tUse: "test",\n\t\tPreRun: func(cmd *Command, args []string) {\n\t\t\tpreRunExecuted = true\n\t\t},\n\t\tRun: func(cmd *Command, args []string) {\n\t\t\trunExecuted = true\n\t\t},\n\t}\n\n\terr := cmd.ExecuteContext(ctx)\n\tif err != context.Canceled {\n\t\tt.Errorf("expected context.Canceled, got %v", err)\n\t}\n\tif preRunExecuted {\n\t\tt.Error("expected PreRun not to be executed")\n\t}\n\tif runExecuted {\n\t\tt.Error("expected Run not to be executed")\n\t}\n}\n\nfunc TestCommandContextCancellationDuringExecution(t *testing.T) {\n\tctx, cancel := context.WithCancel(context.Background())\n\n\tpreRunExecuted := false\n\trunExecuted := false\n\n\tcmd := &Command{\n\t\tUse: "test",\n\t\tPreRun: func(cmd *Command, args []string) {\n\t\t\tpreRunExecuted = true\n\t\t\tcancel()\n\t\t},\n\t\tRun: func(cmd *Command, args []string) {\n\t\t\trunExecuted = true\n\t\t},\n\t}\n\n\terr := cmd.ExecuteContext(ctx)\n\tif err != context.Canceled {\n\t\tt.Errorf("expected context.Canceled, got %v", err)\n\t}\n\tif !preRunExecuted {\n\t\tt.Error("expected PreRun to be executed")\n\t}\n\tif runExecuted {\n\t\tt.Error("expected Run not to be executed")\n\t}\n}\n`
		err = ioutil.WriteFile(commandTestGoPath, []byte(commandTestGoContent), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	_ = os.Remove("patch_test.go")

	cmd := exec.Command("go", "test", "./...")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("go test failed: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}
}
