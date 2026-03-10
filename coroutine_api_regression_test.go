package golua_test

import "testing"

func TestCoroutineAPIRegression_RejectsForgedThreadTables(t *testing.T) {
	source := `
		local fake = { __coroutine_id = 0 }
		local ok1, err1 = pcall(coroutine.status, fake)
		local ok2, err2 = pcall(coroutine.resume, fake)
		local ok3, err3 = pcall(coroutine.close, fake)
		assert(ok1 == false and tostring(err1):find("thread expected, got table", 1, true), tostring(err1))
		assert(ok2 == false and tostring(err2):find("thread expected, got table", 1, true), tostring(err2))
		assert(ok3 == false and tostring(err3):find("thread expected, got table", 1, true), tostring(err3))
	`
	runLuaSource(t, source, "test_coroutine_reject_forged_thread_table")
}

func TestCoroutineAPIRegression_IsYieldableRemainsTrueAfterClose(t *testing.T) {
	source := `
		local co = coroutine.create(function()
			coroutine.yield("pause")
		end)
		assert(coroutine.resume(co))
		assert(coroutine.close(co))
		assert(coroutine.status(co) == "dead")
		assert(coroutine.isyieldable(co) == true)
	`
	runLuaSource(t, source, "test_coroutine_isyieldable_after_close")
}
