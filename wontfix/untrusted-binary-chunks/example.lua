-- Executing an untrusted BINARY chunk (load with mode "b", or the default "bt"
-- mode auto-detecting a binary signature) is inherently unsafe — in golua AND
-- in reference Lua. The Lua manual (§6.1, load) is explicit:
--
--   "Lua does not check the consistency of binary chunks. Maliciously crafted
--    binary chunks can crash the interpreter."
--
-- A crafted-but-loadable proto can encode, e.g., a numeric-for loop with a huge
-- bound or pathological control flow that runs effectively forever in the VM.
-- No interpreter that executes raw bytecode without a full verifier defends
-- against this, and reference Lua ships no such verifier.

-- What golua DOES guarantee (and hardens beyond a naive undumper): LOADING a
-- malformed binary chunk is always a CATCHABLE error, never an uncatchable
-- host crash. A corrupt element count can no longer drive an unbounded
-- allocation (see compiler/undump.go readCount); it surfaces as a normal error:
print(load("\27Lua\84" .. ("\0"):rep(40), "evil", "b"))
--> golua:  nil   evil: bad binary format (version mismatch)
-- (a catchable nil+message return — the host process stays alive)

-- What is NOT guaranteed: that EXECUTING an arbitrary crafted chunk terminates
-- or stays memory-safe. Mitigation for sandboxed embeddings: do not expose
-- binary loading to untrusted code — restrict load() to text mode ("t").
