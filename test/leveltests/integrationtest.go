package leveltests

import (
	"fmt"
	"path"
	"testing"

	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/render/rendertest/testbuilder"
	"github.com/smartystreets/goconvey/convey"
)

type LevelChoice int
type ModeChoice int

var (
	Level1 LevelChoice = 1

	ModePassNPlay ModeChoice = 1
)

type Tester interface {
	ValidateExpectations(testcase string)
}

type renderingTester struct {
	lvl         LevelChoice
	renderQueue render.RenderQueueInterface
}

func testdataDir(lvl LevelChoice) string {
	return map[LevelChoice]string{
		Level1: "level1",
	}[lvl]
}

func testLabel(lvl LevelChoice) string {
	return map[LevelChoice]string{
		Level1: "Level 01",
	}[lvl]
}

func (rt *renderingTester) ValidateExpectations(testcase string) {
	expectedFile := rendertest.NewTestdataReference(path.Join(testdataDir(rt.lvl), testcase))

	convey.So(rt.renderQueue, rendertest.ShouldLookLikeFile, expectedFile)
}

func IntegrationTest(t *testing.T, level LevelChoice, mode ModeChoice, fn func(Tester)) {
	testname := fmt.Sprintf("%s end-to-end test", testLabel(level))
	testbuilder.Run(func(renderQueue render.RenderQueueInterface) {
		convey.Convey(testname, t, func(conveyContext convey.C) {
			tst := &renderingTester{
				lvl:         level,
				renderQueue: renderQueue,
			}
			fn(tst)
		})
	})
}
