-- Test os.exit(true) terminates (true = exit code 0)
os.exit(true)
error("SHOULD NOT REACH HERE")
