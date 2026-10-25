window.BENCHMARK_DATA = {
  "lastUpdate": 1792903913704,
  "repoUrl": "https://github.com/sanbei101/im",
  "entries": {
    "GoBench": [
      {
        "commit": {
          "author": {
            "email": "sanbei101@outlook.com",
            "name": "sanbei101",
            "username": "sanbei101"
          },
          "committer": {
            "email": "sanbei101@outlook.com",
            "name": "sanbei101",
            "username": "sanbei101"
          },
          "distinct": true,
          "id": "dd11b0b8b67a41ef29d801f9dc3d3a899a866844",
          "message": "Fix benchmark state leak across -count rounds and quiet designed drops\n\nb.Loop benchmarks run their outer function once while b.Run callbacks\nre-execute per -count round, so the fanout benchmarks registered a fresh\nset of clients on the shared Gateway every round. Stale clients stayed\nin the room registry and session table forever, their send buffers\nfilled, and every broadcast after the first round logged an error per\nstale client: 4.1M error lines in CI, a several-hundred-MB bench.txt\nthat crashed github-action-benchmark with \"Invalid string length\", and\na benchmark measuring error logging instead of fanout. Build the\nfanout benchmarks' Gateway inside the callback so every round starts\nclean.\n\nDowngrade the five sendFrame-failure logs to debug level: dropping a\nframe for a stalled or closed client is the designed backpressure path\n(clients reconcile through room_seq pull), not an error; at error level\none stuck client floods the log in production exactly as it did in CI.",
          "timestamp": "2026-10-25T12:36:21+08:00",
          "tree_id": "90520e11c7c55cb7810438f4e5a369efe07060d0",
          "url": "https://github.com/sanbei101/im/commit/dd11b0b8b67a41ef29d801f9dc3d3a899a866844"
        },
        "date": 1792903913158,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api)",
            "value": 22023,
            "unit": "ns/op\t   26223 B/op\t     152 allocs/op",
            "extra": "54228 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 22023,
            "unit": "ns/op",
            "extra": "54228 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - B/op",
            "value": 26223,
            "unit": "B/op",
            "extra": "54228 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 152,
            "unit": "allocs/op",
            "extra": "54228 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api)",
            "value": 23382,
            "unit": "ns/op\t   26188 B/op\t     152 allocs/op",
            "extra": "52434 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 23382,
            "unit": "ns/op",
            "extra": "52434 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - B/op",
            "value": 26188,
            "unit": "B/op",
            "extra": "52434 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 152,
            "unit": "allocs/op",
            "extra": "52434 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api)",
            "value": 22317,
            "unit": "ns/op\t   26190 B/op\t     152 allocs/op",
            "extra": "52939 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 22317,
            "unit": "ns/op",
            "extra": "52939 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - B/op",
            "value": 26190,
            "unit": "B/op",
            "extra": "52939 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 152,
            "unit": "allocs/op",
            "extra": "52939 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api)",
            "value": 22482,
            "unit": "ns/op\t   26192 B/op\t     152 allocs/op",
            "extra": "51778 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 22482,
            "unit": "ns/op",
            "extra": "51778 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - B/op",
            "value": 26192,
            "unit": "B/op",
            "extra": "51778 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 152,
            "unit": "allocs/op",
            "extra": "51778 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api)",
            "value": 22708,
            "unit": "ns/op\t   26188 B/op\t     152 allocs/op",
            "extra": "52056 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 22708,
            "unit": "ns/op",
            "extra": "52056 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - B/op",
            "value": 26188,
            "unit": "B/op",
            "extra": "52056 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 152,
            "unit": "allocs/op",
            "extra": "52056 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api)",
            "value": 22500,
            "unit": "ns/op\t   26187 B/op\t     152 allocs/op",
            "extra": "52704 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 22500,
            "unit": "ns/op",
            "extra": "52704 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - B/op",
            "value": 26187,
            "unit": "B/op",
            "extra": "52704 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 152,
            "unit": "allocs/op",
            "extra": "52704 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api)",
            "value": 22607,
            "unit": "ns/op\t   26189 B/op\t     152 allocs/op",
            "extra": "52198 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 22607,
            "unit": "ns/op",
            "extra": "52198 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - B/op",
            "value": 26189,
            "unit": "B/op",
            "extra": "52198 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 152,
            "unit": "allocs/op",
            "extra": "52198 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api)",
            "value": 22369,
            "unit": "ns/op\t   26188 B/op\t     152 allocs/op",
            "extra": "52821 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 22369,
            "unit": "ns/op",
            "extra": "52821 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - B/op",
            "value": 26188,
            "unit": "B/op",
            "extra": "52821 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 152,
            "unit": "allocs/op",
            "extra": "52821 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api)",
            "value": 22964,
            "unit": "ns/op\t   26190 B/op\t     152 allocs/op",
            "extra": "50514 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 22964,
            "unit": "ns/op",
            "extra": "50514 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - B/op",
            "value": 26190,
            "unit": "B/op",
            "extra": "50514 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 152,
            "unit": "allocs/op",
            "extra": "50514 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api)",
            "value": 22356,
            "unit": "ns/op\t   26190 B/op\t     152 allocs/op",
            "extra": "52950 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 22356,
            "unit": "ns/op",
            "extra": "52950 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - B/op",
            "value": 26190,
            "unit": "B/op",
            "extra": "52950 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIHistory (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 152,
            "unit": "allocs/op",
            "extra": "52950 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api)",
            "value": 1455185,
            "unit": "ns/op\t   49019 B/op\t     325 allocs/op",
            "extra": "835 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 1455185,
            "unit": "ns/op",
            "extra": "835 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - B/op",
            "value": 49019,
            "unit": "B/op",
            "extra": "835 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 325,
            "unit": "allocs/op",
            "extra": "835 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api)",
            "value": 1499738,
            "unit": "ns/op\t   49092 B/op\t     325 allocs/op",
            "extra": "830 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 1499738,
            "unit": "ns/op",
            "extra": "830 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - B/op",
            "value": 49092,
            "unit": "B/op",
            "extra": "830 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 325,
            "unit": "allocs/op",
            "extra": "830 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api)",
            "value": 1487370,
            "unit": "ns/op\t   49000 B/op\t     325 allocs/op",
            "extra": "843 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 1487370,
            "unit": "ns/op",
            "extra": "843 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - B/op",
            "value": 49000,
            "unit": "B/op",
            "extra": "843 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 325,
            "unit": "allocs/op",
            "extra": "843 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api)",
            "value": 1449843,
            "unit": "ns/op\t   49046 B/op\t     325 allocs/op",
            "extra": "834 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 1449843,
            "unit": "ns/op",
            "extra": "834 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - B/op",
            "value": 49046,
            "unit": "B/op",
            "extra": "834 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 325,
            "unit": "allocs/op",
            "extra": "834 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api)",
            "value": 1456980,
            "unit": "ns/op\t   48949 B/op\t     325 allocs/op",
            "extra": "846 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 1456980,
            "unit": "ns/op",
            "extra": "846 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - B/op",
            "value": 48949,
            "unit": "B/op",
            "extra": "846 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 325,
            "unit": "allocs/op",
            "extra": "846 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api)",
            "value": 1655003,
            "unit": "ns/op\t   48959 B/op\t     325 allocs/op",
            "extra": "842 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 1655003,
            "unit": "ns/op",
            "extra": "842 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - B/op",
            "value": 48959,
            "unit": "B/op",
            "extra": "842 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 325,
            "unit": "allocs/op",
            "extra": "842 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api)",
            "value": 1449013,
            "unit": "ns/op\t   48996 B/op\t     325 allocs/op",
            "extra": "841 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 1449013,
            "unit": "ns/op",
            "extra": "841 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - B/op",
            "value": 48996,
            "unit": "B/op",
            "extra": "841 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 325,
            "unit": "allocs/op",
            "extra": "841 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api)",
            "value": 1439499,
            "unit": "ns/op\t   49092 B/op\t     325 allocs/op",
            "extra": "838 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 1439499,
            "unit": "ns/op",
            "extra": "838 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - B/op",
            "value": 49092,
            "unit": "B/op",
            "extra": "838 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 325,
            "unit": "allocs/op",
            "extra": "838 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api)",
            "value": 1439866,
            "unit": "ns/op\t   48999 B/op\t     325 allocs/op",
            "extra": "838 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 1439866,
            "unit": "ns/op",
            "extra": "838 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - B/op",
            "value": 48999,
            "unit": "B/op",
            "extra": "838 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 325,
            "unit": "allocs/op",
            "extra": "838 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api)",
            "value": 1461942,
            "unit": "ns/op\t   48997 B/op\t     325 allocs/op",
            "extra": "840 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 1461942,
            "unit": "ns/op",
            "extra": "840 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - B/op",
            "value": 48997,
            "unit": "B/op",
            "extra": "840 times\n4 procs"
          },
          {
            "name": "BenchmarkAPISendBatch (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 325,
            "unit": "allocs/op",
            "extra": "840 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api)",
            "value": 273767,
            "unit": "ns/op\t   10797 B/op\t      77 allocs/op",
            "extra": "5210 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 273767,
            "unit": "ns/op",
            "extra": "5210 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - B/op",
            "value": 10797,
            "unit": "B/op",
            "extra": "5210 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 77,
            "unit": "allocs/op",
            "extra": "5210 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api)",
            "value": 277217,
            "unit": "ns/op\t   10779 B/op\t      77 allocs/op",
            "extra": "5403 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 277217,
            "unit": "ns/op",
            "extra": "5403 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - B/op",
            "value": 10779,
            "unit": "B/op",
            "extra": "5403 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 77,
            "unit": "allocs/op",
            "extra": "5403 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api)",
            "value": 270924,
            "unit": "ns/op\t   10782 B/op\t      77 allocs/op",
            "extra": "5430 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 270924,
            "unit": "ns/op",
            "extra": "5430 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - B/op",
            "value": 10782,
            "unit": "B/op",
            "extra": "5430 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 77,
            "unit": "allocs/op",
            "extra": "5430 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api)",
            "value": 405715,
            "unit": "ns/op\t   10776 B/op\t      77 allocs/op",
            "extra": "5590 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 405715,
            "unit": "ns/op",
            "extra": "5590 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - B/op",
            "value": 10776,
            "unit": "B/op",
            "extra": "5590 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 77,
            "unit": "allocs/op",
            "extra": "5590 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api)",
            "value": 275931,
            "unit": "ns/op\t   10776 B/op\t      77 allocs/op",
            "extra": "5476 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 275931,
            "unit": "ns/op",
            "extra": "5476 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - B/op",
            "value": 10776,
            "unit": "B/op",
            "extra": "5476 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 77,
            "unit": "allocs/op",
            "extra": "5476 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api)",
            "value": 278265,
            "unit": "ns/op\t   10766 B/op\t      77 allocs/op",
            "extra": "5626 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 278265,
            "unit": "ns/op",
            "extra": "5626 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - B/op",
            "value": 10766,
            "unit": "B/op",
            "extra": "5626 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 77,
            "unit": "allocs/op",
            "extra": "5626 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api)",
            "value": 282356,
            "unit": "ns/op\t   10770 B/op\t      77 allocs/op",
            "extra": "5461 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 282356,
            "unit": "ns/op",
            "extra": "5461 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - B/op",
            "value": 10770,
            "unit": "B/op",
            "extra": "5461 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 77,
            "unit": "allocs/op",
            "extra": "5461 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api)",
            "value": 273605,
            "unit": "ns/op\t   10766 B/op\t      77 allocs/op",
            "extra": "5083 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 273605,
            "unit": "ns/op",
            "extra": "5083 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - B/op",
            "value": 10766,
            "unit": "B/op",
            "extra": "5083 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 77,
            "unit": "allocs/op",
            "extra": "5083 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api)",
            "value": 273220,
            "unit": "ns/op\t   10778 B/op\t      77 allocs/op",
            "extra": "5408 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 273220,
            "unit": "ns/op",
            "extra": "5408 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - B/op",
            "value": 10778,
            "unit": "B/op",
            "extra": "5408 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 77,
            "unit": "allocs/op",
            "extra": "5408 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api)",
            "value": 278904,
            "unit": "ns/op\t   10779 B/op\t      77 allocs/op",
            "extra": "5634 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - ns/op",
            "value": 278904,
            "unit": "ns/op",
            "extra": "5634 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - B/op",
            "value": 10779,
            "unit": "B/op",
            "extra": "5634 times\n4 procs"
          },
          {
            "name": "BenchmarkAPIAddReaction (github.com/sanbei101/im/internal/api) - allocs/op",
            "value": 77,
            "unit": "allocs/op",
            "extra": "5634 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway)",
            "value": 15.63,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "76606700 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 15.63,
            "unit": "ns/op",
            "extra": "76606700 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "76606700 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "76606700 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway)",
            "value": 15.56,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "77036587 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 15.56,
            "unit": "ns/op",
            "extra": "77036587 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "77036587 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "77036587 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway)",
            "value": 15.58,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "77131356 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 15.58,
            "unit": "ns/op",
            "extra": "77131356 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "77131356 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "77131356 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway)",
            "value": 15.6,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "77214262 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 15.6,
            "unit": "ns/op",
            "extra": "77214262 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "77214262 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "77214262 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway)",
            "value": 15.73,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "71676747 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 15.73,
            "unit": "ns/op",
            "extra": "71676747 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "71676747 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "71676747 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway)",
            "value": 15.58,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "77079651 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 15.58,
            "unit": "ns/op",
            "extra": "77079651 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "77079651 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "77079651 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway)",
            "value": 15.56,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "77268187 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 15.56,
            "unit": "ns/op",
            "extra": "77268187 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "77268187 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "77268187 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway)",
            "value": 16,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "77237756 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 16,
            "unit": "ns/op",
            "extra": "77237756 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "77237756 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "77237756 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway)",
            "value": 15.94,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "76842421 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 15.94,
            "unit": "ns/op",
            "extra": "76842421 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "76842421 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "76842421 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway)",
            "value": 15.68,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "77017274 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 15.68,
            "unit": "ns/op",
            "extra": "77017274 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "77017274 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/session-load-or-create (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "77017274 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway)",
            "value": 84.92,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "14638644 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 84.92,
            "unit": "ns/op",
            "extra": "14638644 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "14638644 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "14638644 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway)",
            "value": 82.18,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "14595636 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 82.18,
            "unit": "ns/op",
            "extra": "14595636 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "14595636 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "14595636 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway)",
            "value": 83.11,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "14601076 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 83.11,
            "unit": "ns/op",
            "extra": "14601076 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "14601076 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "14601076 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway)",
            "value": 83.11,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "14388720 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 83.11,
            "unit": "ns/op",
            "extra": "14388720 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "14388720 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "14388720 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway)",
            "value": 83.21,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "14628530 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 83.21,
            "unit": "ns/op",
            "extra": "14628530 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "14628530 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "14628530 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway)",
            "value": 82.46,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "14537946 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 82.46,
            "unit": "ns/op",
            "extra": "14537946 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "14537946 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "14537946 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway)",
            "value": 82.35,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "14620884 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 82.35,
            "unit": "ns/op",
            "extra": "14620884 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "14620884 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "14620884 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway)",
            "value": 82.4,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "14597881 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 82.4,
            "unit": "ns/op",
            "extra": "14597881 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "14597881 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "14597881 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway)",
            "value": 83.78,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "14296316 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 83.78,
            "unit": "ns/op",
            "extra": "14296316 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "14296316 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "14296316 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway)",
            "value": 83.7,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "14376776 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 83.7,
            "unit": "ns/op",
            "extra": "14376776 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "14376776 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/touch-room-user (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "14376776 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway)",
            "value": 206,
            "unit": "ns/op\t      48 B/op\t       1 allocs/op",
            "extra": "5842826 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 206,
            "unit": "ns/op",
            "extra": "5842826 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 48,
            "unit": "B/op",
            "extra": "5842826 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "5842826 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway)",
            "value": 206.4,
            "unit": "ns/op\t      48 B/op\t       1 allocs/op",
            "extra": "5831804 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 206.4,
            "unit": "ns/op",
            "extra": "5831804 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 48,
            "unit": "B/op",
            "extra": "5831804 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "5831804 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway)",
            "value": 205.5,
            "unit": "ns/op\t      48 B/op\t       1 allocs/op",
            "extra": "5831959 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 205.5,
            "unit": "ns/op",
            "extra": "5831959 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 48,
            "unit": "B/op",
            "extra": "5831959 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "5831959 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway)",
            "value": 205.9,
            "unit": "ns/op\t      48 B/op\t       1 allocs/op",
            "extra": "5792594 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 205.9,
            "unit": "ns/op",
            "extra": "5792594 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 48,
            "unit": "B/op",
            "extra": "5792594 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "5792594 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway)",
            "value": 205.9,
            "unit": "ns/op\t      48 B/op\t       1 allocs/op",
            "extra": "5850242 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 205.9,
            "unit": "ns/op",
            "extra": "5850242 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 48,
            "unit": "B/op",
            "extra": "5850242 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "5850242 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway)",
            "value": 206.5,
            "unit": "ns/op\t      48 B/op\t       1 allocs/op",
            "extra": "5847237 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 206.5,
            "unit": "ns/op",
            "extra": "5847237 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 48,
            "unit": "B/op",
            "extra": "5847237 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "5847237 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway)",
            "value": 205.2,
            "unit": "ns/op\t      48 B/op\t       1 allocs/op",
            "extra": "5827032 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 205.2,
            "unit": "ns/op",
            "extra": "5827032 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 48,
            "unit": "B/op",
            "extra": "5827032 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "5827032 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway)",
            "value": 205.4,
            "unit": "ns/op\t      48 B/op\t       1 allocs/op",
            "extra": "5831400 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 205.4,
            "unit": "ns/op",
            "extra": "5831400 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 48,
            "unit": "B/op",
            "extra": "5831400 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "5831400 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway)",
            "value": 206.3,
            "unit": "ns/op\t      48 B/op\t       1 allocs/op",
            "extra": "5815369 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 206.3,
            "unit": "ns/op",
            "extra": "5815369 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 48,
            "unit": "B/op",
            "extra": "5815369 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "5815369 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway)",
            "value": 208.3,
            "unit": "ns/op\t      48 B/op\t       1 allocs/op",
            "extra": "5675516 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 208.3,
            "unit": "ns/op",
            "extra": "5675516 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 48,
            "unit": "B/op",
            "extra": "5675516 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/pending-round-trip (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "5675516 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway)",
            "value": 5388,
            "unit": "ns/op\t    2464 B/op\t      11 allocs/op",
            "extra": "219514 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 5388,
            "unit": "ns/op",
            "extra": "219514 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 2464,
            "unit": "B/op",
            "extra": "219514 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "219514 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway)",
            "value": 5402,
            "unit": "ns/op\t    2464 B/op\t      11 allocs/op",
            "extra": "222351 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 5402,
            "unit": "ns/op",
            "extra": "222351 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 2464,
            "unit": "B/op",
            "extra": "222351 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "222351 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway)",
            "value": 5411,
            "unit": "ns/op\t    2464 B/op\t      11 allocs/op",
            "extra": "219572 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 5411,
            "unit": "ns/op",
            "extra": "219572 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 2464,
            "unit": "B/op",
            "extra": "219572 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "219572 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway)",
            "value": 5458,
            "unit": "ns/op\t    2464 B/op\t      11 allocs/op",
            "extra": "219369 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 5458,
            "unit": "ns/op",
            "extra": "219369 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 2464,
            "unit": "B/op",
            "extra": "219369 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "219369 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway)",
            "value": 5414,
            "unit": "ns/op\t    2464 B/op\t      11 allocs/op",
            "extra": "222612 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 5414,
            "unit": "ns/op",
            "extra": "222612 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 2464,
            "unit": "B/op",
            "extra": "222612 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "222612 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway)",
            "value": 5409,
            "unit": "ns/op\t    2464 B/op\t      11 allocs/op",
            "extra": "219805 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 5409,
            "unit": "ns/op",
            "extra": "219805 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 2464,
            "unit": "B/op",
            "extra": "219805 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "219805 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway)",
            "value": 5379,
            "unit": "ns/op\t    2464 B/op\t      11 allocs/op",
            "extra": "220568 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 5379,
            "unit": "ns/op",
            "extra": "220568 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 2464,
            "unit": "B/op",
            "extra": "220568 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "220568 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway)",
            "value": 5440,
            "unit": "ns/op\t    2464 B/op\t      11 allocs/op",
            "extra": "220293 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 5440,
            "unit": "ns/op",
            "extra": "220293 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 2464,
            "unit": "B/op",
            "extra": "220293 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "220293 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway)",
            "value": 5412,
            "unit": "ns/op\t    2464 B/op\t      11 allocs/op",
            "extra": "218043 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 5412,
            "unit": "ns/op",
            "extra": "218043 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 2464,
            "unit": "B/op",
            "extra": "218043 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "218043 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway)",
            "value": 5400,
            "unit": "ns/op\t    2464 B/op\t      11 allocs/op",
            "extra": "219349 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 5400,
            "unit": "ns/op",
            "extra": "219349 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 2464,
            "unit": "B/op",
            "extra": "219349 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/broadcast-typing (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "219349 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway)",
            "value": 6110,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "191974 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 6110,
            "unit": "ns/op",
            "extra": "191974 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "191974 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "191974 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway)",
            "value": 6107,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "193089 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 6107,
            "unit": "ns/op",
            "extra": "193089 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "193089 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "193089 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway)",
            "value": 6204,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "193552 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 6204,
            "unit": "ns/op",
            "extra": "193552 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "193552 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "193552 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway)",
            "value": 6123,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "191394 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 6123,
            "unit": "ns/op",
            "extra": "191394 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "191394 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "191394 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway)",
            "value": 6117,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "195063 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 6117,
            "unit": "ns/op",
            "extra": "195063 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "195063 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "195063 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway)",
            "value": 6157,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "189157 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 6157,
            "unit": "ns/op",
            "extra": "189157 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "189157 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "189157 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway)",
            "value": 6203,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "194788 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 6203,
            "unit": "ns/op",
            "extra": "194788 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "194788 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "194788 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway)",
            "value": 6100,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "192498 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 6100,
            "unit": "ns/op",
            "extra": "192498 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "192498 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "192498 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway)",
            "value": 6125,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "196213 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 6125,
            "unit": "ns/op",
            "extra": "196213 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "196213 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "196213 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway)",
            "value": 6104,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "191884 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - ns/op",
            "value": 6104,
            "unit": "ns/op",
            "extra": "191884 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "191884 times\n4 procs"
          },
          {
            "name": "BenchmarkGateway/room-push-fanout (github.com/sanbei101/im/internal/gateway) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "191884 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router)",
            "value": 46058,
            "unit": "ns/op\t   45783 B/op\t      93 allocs/op",
            "extra": "25782 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 46058,
            "unit": "ns/op",
            "extra": "25782 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - B/op",
            "value": 45783,
            "unit": "B/op",
            "extra": "25782 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 93,
            "unit": "allocs/op",
            "extra": "25782 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router)",
            "value": 44604,
            "unit": "ns/op\t   45840 B/op\t      93 allocs/op",
            "extra": "28789 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 44604,
            "unit": "ns/op",
            "extra": "28789 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - B/op",
            "value": 45840,
            "unit": "B/op",
            "extra": "28789 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 93,
            "unit": "allocs/op",
            "extra": "28789 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router)",
            "value": 46269,
            "unit": "ns/op\t   45845 B/op\t      93 allocs/op",
            "extra": "25990 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 46269,
            "unit": "ns/op",
            "extra": "25990 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - B/op",
            "value": 45845,
            "unit": "B/op",
            "extra": "25990 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 93,
            "unit": "allocs/op",
            "extra": "25990 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router)",
            "value": 46637,
            "unit": "ns/op\t   45835 B/op\t      93 allocs/op",
            "extra": "25972 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 46637,
            "unit": "ns/op",
            "extra": "25972 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - B/op",
            "value": 45835,
            "unit": "B/op",
            "extra": "25972 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 93,
            "unit": "allocs/op",
            "extra": "25972 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router)",
            "value": 47779,
            "unit": "ns/op\t   45783 B/op\t      93 allocs/op",
            "extra": "25124 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 47779,
            "unit": "ns/op",
            "extra": "25124 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - B/op",
            "value": 45783,
            "unit": "B/op",
            "extra": "25124 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 93,
            "unit": "allocs/op",
            "extra": "25124 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router)",
            "value": 46725,
            "unit": "ns/op\t   45873 B/op\t      93 allocs/op",
            "extra": "25417 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 46725,
            "unit": "ns/op",
            "extra": "25417 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - B/op",
            "value": 45873,
            "unit": "B/op",
            "extra": "25417 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 93,
            "unit": "allocs/op",
            "extra": "25417 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router)",
            "value": 46145,
            "unit": "ns/op\t   45803 B/op\t      93 allocs/op",
            "extra": "25848 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 46145,
            "unit": "ns/op",
            "extra": "25848 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - B/op",
            "value": 45803,
            "unit": "B/op",
            "extra": "25848 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 93,
            "unit": "allocs/op",
            "extra": "25848 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router)",
            "value": 46004,
            "unit": "ns/op\t   45879 B/op\t      93 allocs/op",
            "extra": "26066 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 46004,
            "unit": "ns/op",
            "extra": "26066 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - B/op",
            "value": 45879,
            "unit": "B/op",
            "extra": "26066 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 93,
            "unit": "allocs/op",
            "extra": "26066 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router)",
            "value": 46174,
            "unit": "ns/op\t   45751 B/op\t      93 allocs/op",
            "extra": "26037 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 46174,
            "unit": "ns/op",
            "extra": "26037 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - B/op",
            "value": 45751,
            "unit": "B/op",
            "extra": "26037 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 93,
            "unit": "allocs/op",
            "extra": "26037 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router)",
            "value": 46309,
            "unit": "ns/op\t   45820 B/op\t      93 allocs/op",
            "extra": "25767 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 46309,
            "unit": "ns/op",
            "extra": "25767 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - B/op",
            "value": 45820,
            "unit": "B/op",
            "extra": "25767 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/room-post-path (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 93,
            "unit": "allocs/op",
            "extra": "25767 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router)",
            "value": 1253,
            "unit": "ns/op\t    5521 B/op\t      17 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 1253,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - B/op",
            "value": 5521,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router)",
            "value": 1252,
            "unit": "ns/op\t    5521 B/op\t      17 allocs/op",
            "extra": "969798 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 1252,
            "unit": "ns/op",
            "extra": "969798 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - B/op",
            "value": 5521,
            "unit": "B/op",
            "extra": "969798 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "969798 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router)",
            "value": 1275,
            "unit": "ns/op\t    5521 B/op\t      17 allocs/op",
            "extra": "934869 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 1275,
            "unit": "ns/op",
            "extra": "934869 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - B/op",
            "value": 5521,
            "unit": "B/op",
            "extra": "934869 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "934869 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router)",
            "value": 1261,
            "unit": "ns/op\t    5521 B/op\t      17 allocs/op",
            "extra": "911098 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 1261,
            "unit": "ns/op",
            "extra": "911098 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - B/op",
            "value": 5521,
            "unit": "B/op",
            "extra": "911098 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "911098 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router)",
            "value": 1221,
            "unit": "ns/op\t    5521 B/op\t      17 allocs/op",
            "extra": "825374 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 1221,
            "unit": "ns/op",
            "extra": "825374 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - B/op",
            "value": 5521,
            "unit": "B/op",
            "extra": "825374 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "825374 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router)",
            "value": 1244,
            "unit": "ns/op\t    5521 B/op\t      17 allocs/op",
            "extra": "902694 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 1244,
            "unit": "ns/op",
            "extra": "902694 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - B/op",
            "value": 5521,
            "unit": "B/op",
            "extra": "902694 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "902694 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router)",
            "value": 1225,
            "unit": "ns/op\t    5521 B/op\t      17 allocs/op",
            "extra": "965880 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 1225,
            "unit": "ns/op",
            "extra": "965880 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - B/op",
            "value": 5521,
            "unit": "B/op",
            "extra": "965880 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "965880 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router)",
            "value": 1272,
            "unit": "ns/op\t    5521 B/op\t      17 allocs/op",
            "extra": "983048 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 1272,
            "unit": "ns/op",
            "extra": "983048 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - B/op",
            "value": 5521,
            "unit": "B/op",
            "extra": "983048 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "983048 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router)",
            "value": 1220,
            "unit": "ns/op\t    5521 B/op\t      17 allocs/op",
            "extra": "955119 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 1220,
            "unit": "ns/op",
            "extra": "955119 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - B/op",
            "value": 5521,
            "unit": "B/op",
            "extra": "955119 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "955119 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router)",
            "value": 1209,
            "unit": "ns/op\t    5521 B/op\t      17 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - ns/op",
            "value": 1209,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - B/op",
            "value": 5521,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEdgeRoute/plain-post-body (github.com/sanbei101/im/internal/router) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store)",
            "value": 240862,
            "unit": "ns/op\t  118405 B/op\t    1215 allocs/op",
            "extra": "4286 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 240862,
            "unit": "ns/op",
            "extra": "4286 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118405,
            "unit": "B/op",
            "extra": "4286 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "4286 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store)",
            "value": 241722,
            "unit": "ns/op\t  118404 B/op\t    1215 allocs/op",
            "extra": "4771 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 241722,
            "unit": "ns/op",
            "extra": "4771 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118404,
            "unit": "B/op",
            "extra": "4771 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "4771 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store)",
            "value": 242234,
            "unit": "ns/op\t  118405 B/op\t    1215 allocs/op",
            "extra": "4723 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 242234,
            "unit": "ns/op",
            "extra": "4723 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118405,
            "unit": "B/op",
            "extra": "4723 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "4723 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store)",
            "value": 237989,
            "unit": "ns/op\t  118404 B/op\t    1215 allocs/op",
            "extra": "4764 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 237989,
            "unit": "ns/op",
            "extra": "4764 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118404,
            "unit": "B/op",
            "extra": "4764 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "4764 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store)",
            "value": 241692,
            "unit": "ns/op\t  118404 B/op\t    1215 allocs/op",
            "extra": "4668 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 241692,
            "unit": "ns/op",
            "extra": "4668 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118404,
            "unit": "B/op",
            "extra": "4668 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "4668 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store)",
            "value": 242655,
            "unit": "ns/op\t  118405 B/op\t    1215 allocs/op",
            "extra": "4687 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 242655,
            "unit": "ns/op",
            "extra": "4687 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118405,
            "unit": "B/op",
            "extra": "4687 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "4687 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store)",
            "value": 241007,
            "unit": "ns/op\t  118405 B/op\t    1215 allocs/op",
            "extra": "4722 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 241007,
            "unit": "ns/op",
            "extra": "4722 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118405,
            "unit": "B/op",
            "extra": "4722 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "4722 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store)",
            "value": 240583,
            "unit": "ns/op\t  118404 B/op\t    1215 allocs/op",
            "extra": "4620 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 240583,
            "unit": "ns/op",
            "extra": "4620 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118404,
            "unit": "B/op",
            "extra": "4620 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "4620 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store)",
            "value": 239186,
            "unit": "ns/op\t  118404 B/op\t    1215 allocs/op",
            "extra": "4653 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 239186,
            "unit": "ns/op",
            "extra": "4653 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118404,
            "unit": "B/op",
            "extra": "4653 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "4653 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store)",
            "value": 241380,
            "unit": "ns/op\t  118404 B/op\t    1215 allocs/op",
            "extra": "4760 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 241380,
            "unit": "ns/op",
            "extra": "4760 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118404,
            "unit": "B/op",
            "extra": "4760 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/serial (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "4760 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store)",
            "value": 142762,
            "unit": "ns/op\t  118437 B/op\t    1215 allocs/op",
            "extra": "8311 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 142762,
            "unit": "ns/op",
            "extra": "8311 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118437,
            "unit": "B/op",
            "extra": "8311 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "8311 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store)",
            "value": 143301,
            "unit": "ns/op\t  118437 B/op\t    1215 allocs/op",
            "extra": "8409 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 143301,
            "unit": "ns/op",
            "extra": "8409 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118437,
            "unit": "B/op",
            "extra": "8409 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "8409 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store)",
            "value": 150135,
            "unit": "ns/op\t  118443 B/op\t    1215 allocs/op",
            "extra": "6829 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 150135,
            "unit": "ns/op",
            "extra": "6829 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118443,
            "unit": "B/op",
            "extra": "6829 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "6829 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store)",
            "value": 147325,
            "unit": "ns/op\t  118445 B/op\t    1215 allocs/op",
            "extra": "6808 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 147325,
            "unit": "ns/op",
            "extra": "6808 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118445,
            "unit": "B/op",
            "extra": "6808 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "6808 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store)",
            "value": 145372,
            "unit": "ns/op\t  118437 B/op\t    1215 allocs/op",
            "extra": "8284 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 145372,
            "unit": "ns/op",
            "extra": "8284 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118437,
            "unit": "B/op",
            "extra": "8284 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "8284 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store)",
            "value": 146168,
            "unit": "ns/op\t  118438 B/op\t    1215 allocs/op",
            "extra": "8180 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 146168,
            "unit": "ns/op",
            "extra": "8180 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118438,
            "unit": "B/op",
            "extra": "8180 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "8180 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store)",
            "value": 144939,
            "unit": "ns/op\t  118439 B/op\t    1215 allocs/op",
            "extra": "8119 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 144939,
            "unit": "ns/op",
            "extra": "8119 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118439,
            "unit": "B/op",
            "extra": "8119 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "8119 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store)",
            "value": 145960,
            "unit": "ns/op\t  118439 B/op\t    1215 allocs/op",
            "extra": "8136 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 145960,
            "unit": "ns/op",
            "extra": "8136 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118439,
            "unit": "B/op",
            "extra": "8136 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "8136 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store)",
            "value": 145431,
            "unit": "ns/op\t  118436 B/op\t    1215 allocs/op",
            "extra": "8337 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 145431,
            "unit": "ns/op",
            "extra": "8337 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118436,
            "unit": "B/op",
            "extra": "8337 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "8337 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store)",
            "value": 147157,
            "unit": "ns/op\t  118443 B/op\t    1215 allocs/op",
            "extra": "6814 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 147157,
            "unit": "ns/op",
            "extra": "6814 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - B/op",
            "value": 118443,
            "unit": "B/op",
            "extra": "6814 times\n4 procs"
          },
          {
            "name": "BenchmarkDBConversationList/parallel (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 1215,
            "unit": "allocs/op",
            "extra": "6814 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store)",
            "value": 390237,
            "unit": "ns/op\t    9100 B/op\t     150 allocs/op",
            "extra": "3058 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 390237,
            "unit": "ns/op",
            "extra": "3058 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - B/op",
            "value": 9100,
            "unit": "B/op",
            "extra": "3058 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 150,
            "unit": "allocs/op",
            "extra": "3058 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store)",
            "value": 391168,
            "unit": "ns/op\t    9099 B/op\t     150 allocs/op",
            "extra": "2989 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 391168,
            "unit": "ns/op",
            "extra": "2989 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - B/op",
            "value": 9099,
            "unit": "B/op",
            "extra": "2989 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 150,
            "unit": "allocs/op",
            "extra": "2989 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store)",
            "value": 391100,
            "unit": "ns/op\t    9100 B/op\t     150 allocs/op",
            "extra": "2931 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 391100,
            "unit": "ns/op",
            "extra": "2931 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - B/op",
            "value": 9100,
            "unit": "B/op",
            "extra": "2931 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 150,
            "unit": "allocs/op",
            "extra": "2931 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store)",
            "value": 417089,
            "unit": "ns/op\t    9100 B/op\t     150 allocs/op",
            "extra": "2792 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 417089,
            "unit": "ns/op",
            "extra": "2792 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - B/op",
            "value": 9100,
            "unit": "B/op",
            "extra": "2792 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 150,
            "unit": "allocs/op",
            "extra": "2792 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store)",
            "value": 391636,
            "unit": "ns/op\t    9099 B/op\t     150 allocs/op",
            "extra": "2990 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 391636,
            "unit": "ns/op",
            "extra": "2990 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - B/op",
            "value": 9099,
            "unit": "B/op",
            "extra": "2990 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 150,
            "unit": "allocs/op",
            "extra": "2990 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store)",
            "value": 390393,
            "unit": "ns/op\t    9099 B/op\t     150 allocs/op",
            "extra": "3008 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 390393,
            "unit": "ns/op",
            "extra": "3008 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - B/op",
            "value": 9099,
            "unit": "B/op",
            "extra": "3008 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 150,
            "unit": "allocs/op",
            "extra": "3008 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store)",
            "value": 393447,
            "unit": "ns/op\t    9100 B/op\t     150 allocs/op",
            "extra": "3076 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 393447,
            "unit": "ns/op",
            "extra": "3076 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - B/op",
            "value": 9100,
            "unit": "B/op",
            "extra": "3076 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 150,
            "unit": "allocs/op",
            "extra": "3076 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store)",
            "value": 399005,
            "unit": "ns/op\t    9099 B/op\t     150 allocs/op",
            "extra": "2996 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 399005,
            "unit": "ns/op",
            "extra": "2996 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - B/op",
            "value": 9099,
            "unit": "B/op",
            "extra": "2996 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 150,
            "unit": "allocs/op",
            "extra": "2996 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store)",
            "value": 391334,
            "unit": "ns/op\t    9100 B/op\t     150 allocs/op",
            "extra": "3020 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 391334,
            "unit": "ns/op",
            "extra": "3020 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - B/op",
            "value": 9100,
            "unit": "B/op",
            "extra": "3020 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 150,
            "unit": "allocs/op",
            "extra": "3020 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store)",
            "value": 243715,
            "unit": "ns/op\t    9100 B/op\t     150 allocs/op",
            "extra": "4423 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 243715,
            "unit": "ns/op",
            "extra": "4423 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - B/op",
            "value": 9100,
            "unit": "B/op",
            "extra": "4423 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/search (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 150,
            "unit": "allocs/op",
            "extra": "4423 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store)",
            "value": 317190,
            "unit": "ns/op\t  191203 B/op\t    2444 allocs/op",
            "extra": "3636 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 317190,
            "unit": "ns/op",
            "extra": "3636 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - B/op",
            "value": 191203,
            "unit": "B/op",
            "extra": "3636 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 2444,
            "unit": "allocs/op",
            "extra": "3636 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store)",
            "value": 316177,
            "unit": "ns/op\t  191326 B/op\t    2444 allocs/op",
            "extra": "3387 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 316177,
            "unit": "ns/op",
            "extra": "3387 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - B/op",
            "value": 191326,
            "unit": "B/op",
            "extra": "3387 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 2444,
            "unit": "allocs/op",
            "extra": "3387 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store)",
            "value": 314083,
            "unit": "ns/op\t  191245 B/op\t    2444 allocs/op",
            "extra": "3484 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 314083,
            "unit": "ns/op",
            "extra": "3484 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - B/op",
            "value": 191245,
            "unit": "B/op",
            "extra": "3484 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 2444,
            "unit": "allocs/op",
            "extra": "3484 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store)",
            "value": 316945,
            "unit": "ns/op\t  191385 B/op\t    2444 allocs/op",
            "extra": "3663 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 316945,
            "unit": "ns/op",
            "extra": "3663 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - B/op",
            "value": 191385,
            "unit": "B/op",
            "extra": "3663 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 2444,
            "unit": "allocs/op",
            "extra": "3663 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store)",
            "value": 312597,
            "unit": "ns/op\t  191395 B/op\t    2444 allocs/op",
            "extra": "3699 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 312597,
            "unit": "ns/op",
            "extra": "3699 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - B/op",
            "value": 191395,
            "unit": "B/op",
            "extra": "3699 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 2444,
            "unit": "allocs/op",
            "extra": "3699 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store)",
            "value": 320912,
            "unit": "ns/op\t  191336 B/op\t    2444 allocs/op",
            "extra": "3732 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 320912,
            "unit": "ns/op",
            "extra": "3732 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - B/op",
            "value": 191336,
            "unit": "B/op",
            "extra": "3732 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 2444,
            "unit": "allocs/op",
            "extra": "3732 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store)",
            "value": 315318,
            "unit": "ns/op\t  191243 B/op\t    2444 allocs/op",
            "extra": "3698 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 315318,
            "unit": "ns/op",
            "extra": "3698 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - B/op",
            "value": 191243,
            "unit": "B/op",
            "extra": "3698 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 2444,
            "unit": "allocs/op",
            "extra": "3698 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store)",
            "value": 315021,
            "unit": "ns/op\t  191287 B/op\t    2444 allocs/op",
            "extra": "3700 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 315021,
            "unit": "ns/op",
            "extra": "3700 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - B/op",
            "value": 191287,
            "unit": "B/op",
            "extra": "3700 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 2444,
            "unit": "allocs/op",
            "extra": "3700 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store)",
            "value": 313860,
            "unit": "ns/op\t  191311 B/op\t    2444 allocs/op",
            "extra": "3666 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 313860,
            "unit": "ns/op",
            "extra": "3666 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - B/op",
            "value": 191311,
            "unit": "B/op",
            "extra": "3666 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 2444,
            "unit": "allocs/op",
            "extra": "3666 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store)",
            "value": 314733,
            "unit": "ns/op\t  191378 B/op\t    2444 allocs/op",
            "extra": "3685 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - ns/op",
            "value": 314733,
            "unit": "ns/op",
            "extra": "3685 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - B/op",
            "value": 191378,
            "unit": "B/op",
            "extra": "3685 times\n4 procs"
          },
          {
            "name": "BenchmarkDBUsers/by-ids (github.com/sanbei101/im/internal/store) - allocs/op",
            "value": 2444,
            "unit": "allocs/op",
            "extra": "3685 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg)",
            "value": 1.633,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "737916682 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - ns/op",
            "value": 1.633,
            "unit": "ns/op",
            "extra": "737916682 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "737916682 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "737916682 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg)",
            "value": 1.656,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "737515797 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - ns/op",
            "value": 1.656,
            "unit": "ns/op",
            "extra": "737515797 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "737515797 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "737515797 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg)",
            "value": 1.763,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "684151744 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - ns/op",
            "value": 1.763,
            "unit": "ns/op",
            "extra": "684151744 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "684151744 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "684151744 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg)",
            "value": 1.64,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "738290830 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - ns/op",
            "value": 1.64,
            "unit": "ns/op",
            "extra": "738290830 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "738290830 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "738290830 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg)",
            "value": 1.755,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "686620656 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - ns/op",
            "value": 1.755,
            "unit": "ns/op",
            "extra": "686620656 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "686620656 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "686620656 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg)",
            "value": 1.627,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "736624066 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - ns/op",
            "value": 1.627,
            "unit": "ns/op",
            "extra": "736624066 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "736624066 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "736624066 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg)",
            "value": 1.752,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "684835668 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - ns/op",
            "value": 1.752,
            "unit": "ns/op",
            "extra": "684835668 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "684835668 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "684835668 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg)",
            "value": 1.631,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "735600436 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - ns/op",
            "value": 1.631,
            "unit": "ns/op",
            "extra": "735600436 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "735600436 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "735600436 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg)",
            "value": 1.767,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "668308190 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - ns/op",
            "value": 1.767,
            "unit": "ns/op",
            "extra": "668308190 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "668308190 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "668308190 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg)",
            "value": 1.627,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "736356464 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - ns/op",
            "value": 1.627,
            "unit": "ns/op",
            "extra": "736356464 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "736356464 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/room-slot (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "736356464 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg)",
            "value": 0.6329,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - ns/op",
            "value": 0.6329,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg)",
            "value": 0.6322,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - ns/op",
            "value": 0.6322,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg)",
            "value": 0.6325,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - ns/op",
            "value": 0.6325,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg)",
            "value": 0.6398,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - ns/op",
            "value": 0.6398,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg)",
            "value": 0.6337,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - ns/op",
            "value": 0.6337,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg)",
            "value": 0.6382,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - ns/op",
            "value": 0.6382,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg)",
            "value": 0.6334,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - ns/op",
            "value": 0.6334,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg)",
            "value": 0.6306,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - ns/op",
            "value": 0.6306,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg)",
            "value": 0.6318,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - ns/op",
            "value": 0.6318,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg)",
            "value": 0.6308,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - ns/op",
            "value": 0.6308,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkShard/node-index (github.com/sanbei101/im/pkg) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 397045,
            "unit": "ns/op\t 182.45 MB/s\t 1803156 B/op\t     347 allocs/op",
            "extra": "3108 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 397045,
            "unit": "ns/op",
            "extra": "3108 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 182.45,
            "unit": "MB/s",
            "extra": "3108 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1803156,
            "unit": "B/op",
            "extra": "3108 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 347,
            "unit": "allocs/op",
            "extra": "3108 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 420106,
            "unit": "ns/op\t 172.43 MB/s\t 1775726 B/op\t     346 allocs/op",
            "extra": "3277 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 420106,
            "unit": "ns/op",
            "extra": "3277 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 172.43,
            "unit": "MB/s",
            "extra": "3277 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1775726,
            "unit": "B/op",
            "extra": "3277 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 346,
            "unit": "allocs/op",
            "extra": "3277 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 420098,
            "unit": "ns/op\t 172.43 MB/s\t 1832786 B/op\t     348 allocs/op",
            "extra": "3006 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 420098,
            "unit": "ns/op",
            "extra": "3006 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 172.43,
            "unit": "MB/s",
            "extra": "3006 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1832786,
            "unit": "B/op",
            "extra": "3006 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 348,
            "unit": "allocs/op",
            "extra": "3006 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 421907,
            "unit": "ns/op\t 171.69 MB/s\t 1823801 B/op\t     348 allocs/op",
            "extra": "2914 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 421907,
            "unit": "ns/op",
            "extra": "2914 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 171.69,
            "unit": "MB/s",
            "extra": "2914 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1823801,
            "unit": "B/op",
            "extra": "2914 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 348,
            "unit": "allocs/op",
            "extra": "2914 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 427994,
            "unit": "ns/op\t 169.25 MB/s\t 1844424 B/op\t     349 allocs/op",
            "extra": "2678 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 427994,
            "unit": "ns/op",
            "extra": "2678 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 169.25,
            "unit": "MB/s",
            "extra": "2678 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1844424,
            "unit": "B/op",
            "extra": "2678 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 349,
            "unit": "allocs/op",
            "extra": "2678 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 406159,
            "unit": "ns/op\t 178.35 MB/s\t 1821381 B/op\t     348 allocs/op",
            "extra": "2916 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 406159,
            "unit": "ns/op",
            "extra": "2916 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 178.35,
            "unit": "MB/s",
            "extra": "2916 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1821381,
            "unit": "B/op",
            "extra": "2916 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 348,
            "unit": "allocs/op",
            "extra": "2916 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 406923,
            "unit": "ns/op\t 178.02 MB/s\t 1829703 B/op\t     348 allocs/op",
            "extra": "3103 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 406923,
            "unit": "ns/op",
            "extra": "3103 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 178.02,
            "unit": "MB/s",
            "extra": "3103 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1829703,
            "unit": "B/op",
            "extra": "3103 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 348,
            "unit": "allocs/op",
            "extra": "3103 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 401285,
            "unit": "ns/op\t 180.52 MB/s\t 1831351 B/op\t     348 allocs/op",
            "extra": "3122 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 401285,
            "unit": "ns/op",
            "extra": "3122 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 180.52,
            "unit": "MB/s",
            "extra": "3122 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1831351,
            "unit": "B/op",
            "extra": "3122 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 348,
            "unit": "allocs/op",
            "extra": "3122 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 402754,
            "unit": "ns/op\t 179.86 MB/s\t 1778128 B/op\t     346 allocs/op",
            "extra": "3109 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 402754,
            "unit": "ns/op",
            "extra": "3109 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 179.86,
            "unit": "MB/s",
            "extra": "3109 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1778128,
            "unit": "B/op",
            "extra": "3109 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 346,
            "unit": "allocs/op",
            "extra": "3109 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 394444,
            "unit": "ns/op\t 183.65 MB/s\t 1819698 B/op\t     348 allocs/op",
            "extra": "2920 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 394444,
            "unit": "ns/op",
            "extra": "2920 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 183.65,
            "unit": "MB/s",
            "extra": "2920 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1819698,
            "unit": "B/op",
            "extra": "2920 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 348,
            "unit": "allocs/op",
            "extra": "2920 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 966703,
            "unit": "ns/op\t  74.93 MB/s\t 1491279 B/op\t   10263 allocs/op",
            "extra": "1214 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 966703,
            "unit": "ns/op",
            "extra": "1214 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 74.93,
            "unit": "MB/s",
            "extra": "1214 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1491279,
            "unit": "B/op",
            "extra": "1214 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10263,
            "unit": "allocs/op",
            "extra": "1214 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 965070,
            "unit": "ns/op\t  75.06 MB/s\t 1486249 B/op\t   10261 allocs/op",
            "extra": "1250 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 965070,
            "unit": "ns/op",
            "extra": "1250 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 75.06,
            "unit": "MB/s",
            "extra": "1250 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1486249,
            "unit": "B/op",
            "extra": "1250 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10261,
            "unit": "allocs/op",
            "extra": "1250 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 975523,
            "unit": "ns/op\t  74.26 MB/s\t 1492841 B/op\t   10262 allocs/op",
            "extra": "1198 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 975523,
            "unit": "ns/op",
            "extra": "1198 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 74.26,
            "unit": "MB/s",
            "extra": "1198 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1492841,
            "unit": "B/op",
            "extra": "1198 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10262,
            "unit": "allocs/op",
            "extra": "1198 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 987348,
            "unit": "ns/op\t  73.37 MB/s\t 1489101 B/op\t   10262 allocs/op",
            "extra": "1248 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 987348,
            "unit": "ns/op",
            "extra": "1248 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 73.37,
            "unit": "MB/s",
            "extra": "1248 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1489101,
            "unit": "B/op",
            "extra": "1248 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10262,
            "unit": "allocs/op",
            "extra": "1248 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 967354,
            "unit": "ns/op\t  74.88 MB/s\t 1492176 B/op\t   10262 allocs/op",
            "extra": "1236 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 967354,
            "unit": "ns/op",
            "extra": "1236 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 74.88,
            "unit": "MB/s",
            "extra": "1236 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1492176,
            "unit": "B/op",
            "extra": "1236 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10262,
            "unit": "allocs/op",
            "extra": "1236 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 959838,
            "unit": "ns/op\t  75.47 MB/s\t 1490446 B/op\t   10262 allocs/op",
            "extra": "1267 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 959838,
            "unit": "ns/op",
            "extra": "1267 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 75.47,
            "unit": "MB/s",
            "extra": "1267 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1490446,
            "unit": "B/op",
            "extra": "1267 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10262,
            "unit": "allocs/op",
            "extra": "1267 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 965005,
            "unit": "ns/op\t  75.07 MB/s\t 1491224 B/op\t   10262 allocs/op",
            "extra": "1243 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 965005,
            "unit": "ns/op",
            "extra": "1243 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 75.07,
            "unit": "MB/s",
            "extra": "1243 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1491224,
            "unit": "B/op",
            "extra": "1243 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10262,
            "unit": "allocs/op",
            "extra": "1243 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 975948,
            "unit": "ns/op\t  74.22 MB/s\t 1490264 B/op\t   10262 allocs/op",
            "extra": "1255 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 975948,
            "unit": "ns/op",
            "extra": "1255 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 74.22,
            "unit": "MB/s",
            "extra": "1255 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1490264,
            "unit": "B/op",
            "extra": "1255 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10262,
            "unit": "allocs/op",
            "extra": "1255 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 967800,
            "unit": "ns/op\t  74.85 MB/s\t 1495942 B/op\t   10263 allocs/op",
            "extra": "1220 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 967800,
            "unit": "ns/op",
            "extra": "1220 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 74.85,
            "unit": "MB/s",
            "extra": "1220 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1495942,
            "unit": "B/op",
            "extra": "1220 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10263,
            "unit": "allocs/op",
            "extra": "1220 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 963486,
            "unit": "ns/op\t  75.18 MB/s\t 1494522 B/op\t   10262 allocs/op",
            "extra": "1288 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 963486,
            "unit": "ns/op",
            "extra": "1288 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - MB/s",
            "value": 75.18,
            "unit": "MB/s",
            "extra": "1288 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1494522,
            "unit": "B/op",
            "extra": "1288 times\n4 procs"
          },
          {
            "name": "BenchmarkArchiveParquetCodec/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10262,
            "unit": "allocs/op",
            "extra": "1288 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine)",
            "value": 4392,
            "unit": "ns/op\t   14168 B/op\t      20 allocs/op",
            "extra": "277362 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 4392,
            "unit": "ns/op",
            "extra": "277362 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 14168,
            "unit": "B/op",
            "extra": "277362 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "277362 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine)",
            "value": 4582,
            "unit": "ns/op\t   14163 B/op\t      20 allocs/op",
            "extra": "296979 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 4582,
            "unit": "ns/op",
            "extra": "296979 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 14163,
            "unit": "B/op",
            "extra": "296979 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "296979 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine)",
            "value": 4340,
            "unit": "ns/op\t   14163 B/op\t      20 allocs/op",
            "extra": "286844 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 4340,
            "unit": "ns/op",
            "extra": "286844 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 14163,
            "unit": "B/op",
            "extra": "286844 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "286844 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine)",
            "value": 4430,
            "unit": "ns/op\t   14164 B/op\t      20 allocs/op",
            "extra": "276240 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 4430,
            "unit": "ns/op",
            "extra": "276240 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 14164,
            "unit": "B/op",
            "extra": "276240 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "276240 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine)",
            "value": 4428,
            "unit": "ns/op\t   14163 B/op\t      20 allocs/op",
            "extra": "297018 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 4428,
            "unit": "ns/op",
            "extra": "297018 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 14163,
            "unit": "B/op",
            "extra": "297018 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "297018 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine)",
            "value": 4445,
            "unit": "ns/op\t   14163 B/op\t      20 allocs/op",
            "extra": "285776 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 4445,
            "unit": "ns/op",
            "extra": "285776 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 14163,
            "unit": "B/op",
            "extra": "285776 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "285776 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine)",
            "value": 4824,
            "unit": "ns/op\t   14164 B/op\t      20 allocs/op",
            "extra": "231588 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 4824,
            "unit": "ns/op",
            "extra": "231588 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 14164,
            "unit": "B/op",
            "extra": "231588 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "231588 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine)",
            "value": 4556,
            "unit": "ns/op\t   14164 B/op\t      20 allocs/op",
            "extra": "255661 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 4556,
            "unit": "ns/op",
            "extra": "255661 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 14164,
            "unit": "B/op",
            "extra": "255661 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "255661 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine)",
            "value": 4864,
            "unit": "ns/op\t   14163 B/op\t      20 allocs/op",
            "extra": "288382 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 4864,
            "unit": "ns/op",
            "extra": "288382 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 14163,
            "unit": "B/op",
            "extra": "288382 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "288382 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine)",
            "value": 4291,
            "unit": "ns/op\t   14163 B/op\t      20 allocs/op",
            "extra": "298104 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 4291,
            "unit": "ns/op",
            "extra": "298104 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 14163,
            "unit": "B/op",
            "extra": "298104 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "298104 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 2836,
            "unit": "ns/op\t     816 B/op\t      15 allocs/op",
            "extra": "471877 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2836,
            "unit": "ns/op",
            "extra": "471877 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 816,
            "unit": "B/op",
            "extra": "471877 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "471877 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 2767,
            "unit": "ns/op\t     816 B/op\t      15 allocs/op",
            "extra": "430846 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2767,
            "unit": "ns/op",
            "extra": "430846 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 816,
            "unit": "B/op",
            "extra": "430846 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "430846 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 2895,
            "unit": "ns/op\t     816 B/op\t      15 allocs/op",
            "extra": "423427 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2895,
            "unit": "ns/op",
            "extra": "423427 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 816,
            "unit": "B/op",
            "extra": "423427 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "423427 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 2783,
            "unit": "ns/op\t     816 B/op\t      15 allocs/op",
            "extra": "408930 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2783,
            "unit": "ns/op",
            "extra": "408930 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 816,
            "unit": "B/op",
            "extra": "408930 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "408930 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 2926,
            "unit": "ns/op\t     816 B/op\t      15 allocs/op",
            "extra": "393890 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2926,
            "unit": "ns/op",
            "extra": "393890 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 816,
            "unit": "B/op",
            "extra": "393890 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "393890 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 2836,
            "unit": "ns/op\t     816 B/op\t      15 allocs/op",
            "extra": "470607 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2836,
            "unit": "ns/op",
            "extra": "470607 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 816,
            "unit": "B/op",
            "extra": "470607 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "470607 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 2911,
            "unit": "ns/op\t     816 B/op\t      15 allocs/op",
            "extra": "445062 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2911,
            "unit": "ns/op",
            "extra": "445062 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 816,
            "unit": "B/op",
            "extra": "445062 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "445062 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 2807,
            "unit": "ns/op\t     816 B/op\t      15 allocs/op",
            "extra": "438105 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2807,
            "unit": "ns/op",
            "extra": "438105 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 816,
            "unit": "B/op",
            "extra": "438105 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "438105 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 2727,
            "unit": "ns/op\t     816 B/op\t      15 allocs/op",
            "extra": "468781 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2727,
            "unit": "ns/op",
            "extra": "468781 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 816,
            "unit": "B/op",
            "extra": "468781 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "468781 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 2953,
            "unit": "ns/op\t     816 B/op\t      15 allocs/op",
            "extra": "451417 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2953,
            "unit": "ns/op",
            "extra": "451417 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 816,
            "unit": "B/op",
            "extra": "451417 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineArchivedRead/by-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "451417 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine)",
            "value": 37.61,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "31762690 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 37.61,
            "unit": "ns/op",
            "extra": "31762690 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "31762690 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "31762690 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine)",
            "value": 37.52,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "32052678 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 37.52,
            "unit": "ns/op",
            "extra": "32052678 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "32052678 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "32052678 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine)",
            "value": 37.62,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "32044902 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 37.62,
            "unit": "ns/op",
            "extra": "32044902 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "32044902 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "32044902 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine)",
            "value": 37.57,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "31919209 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 37.57,
            "unit": "ns/op",
            "extra": "31919209 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "31919209 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "31919209 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine)",
            "value": 37.63,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "32050258 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 37.63,
            "unit": "ns/op",
            "extra": "32050258 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "32050258 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "32050258 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine)",
            "value": 37.43,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "31789558 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 37.43,
            "unit": "ns/op",
            "extra": "31789558 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "31789558 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "31789558 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine)",
            "value": 37.46,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "32027901 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 37.46,
            "unit": "ns/op",
            "extra": "32027901 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "32027901 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "32027901 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine)",
            "value": 37.67,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "32068550 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 37.67,
            "unit": "ns/op",
            "extra": "32068550 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "32068550 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "32068550 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine)",
            "value": 37.46,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "32050630 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 37.46,
            "unit": "ns/op",
            "extra": "32050630 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "32050630 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "32050630 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine)",
            "value": 37.61,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "32052243 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 37.61,
            "unit": "ns/op",
            "extra": "32052243 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "32052243 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/append (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "32052243 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine)",
            "value": 438.7,
            "unit": "ns/op\t    3072 B/op\t       1 allocs/op",
            "extra": "2757631 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 438.7,
            "unit": "ns/op",
            "extra": "2757631 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3072,
            "unit": "B/op",
            "extra": "2757631 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2757631 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine)",
            "value": 454.8,
            "unit": "ns/op\t    3072 B/op\t       1 allocs/op",
            "extra": "2641446 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 454.8,
            "unit": "ns/op",
            "extra": "2641446 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3072,
            "unit": "B/op",
            "extra": "2641446 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2641446 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine)",
            "value": 441.3,
            "unit": "ns/op\t    3072 B/op\t       1 allocs/op",
            "extra": "2759911 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 441.3,
            "unit": "ns/op",
            "extra": "2759911 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3072,
            "unit": "B/op",
            "extra": "2759911 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2759911 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine)",
            "value": 441.7,
            "unit": "ns/op\t    3072 B/op\t       1 allocs/op",
            "extra": "2670006 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 441.7,
            "unit": "ns/op",
            "extra": "2670006 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3072,
            "unit": "B/op",
            "extra": "2670006 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2670006 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine)",
            "value": 431.7,
            "unit": "ns/op\t    3072 B/op\t       1 allocs/op",
            "extra": "2729176 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 431.7,
            "unit": "ns/op",
            "extra": "2729176 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3072,
            "unit": "B/op",
            "extra": "2729176 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2729176 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine)",
            "value": 440,
            "unit": "ns/op\t    3072 B/op\t       1 allocs/op",
            "extra": "2830099 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 440,
            "unit": "ns/op",
            "extra": "2830099 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3072,
            "unit": "B/op",
            "extra": "2830099 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2830099 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine)",
            "value": 460.4,
            "unit": "ns/op\t    3072 B/op\t       1 allocs/op",
            "extra": "2475643 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 460.4,
            "unit": "ns/op",
            "extra": "2475643 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3072,
            "unit": "B/op",
            "extra": "2475643 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2475643 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine)",
            "value": 452.9,
            "unit": "ns/op\t    3072 B/op\t       1 allocs/op",
            "extra": "2849296 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 452.9,
            "unit": "ns/op",
            "extra": "2849296 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3072,
            "unit": "B/op",
            "extra": "2849296 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2849296 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine)",
            "value": 444.7,
            "unit": "ns/op\t    3072 B/op\t       1 allocs/op",
            "extra": "2711252 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 444.7,
            "unit": "ns/op",
            "extra": "2711252 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3072,
            "unit": "B/op",
            "extra": "2711252 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2711252 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine)",
            "value": 434.4,
            "unit": "ns/op\t    3072 B/op\t       1 allocs/op",
            "extra": "2721096 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 434.4,
            "unit": "ns/op",
            "extra": "2721096 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3072,
            "unit": "B/op",
            "extra": "2721096 times\n4 procs"
          },
          {
            "name": "BenchmarkMessageCache/page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2721096 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine)",
            "value": 1293638,
            "unit": "ns/op\t    2290 B/op\t      17 allocs/op",
            "extra": "931 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1293638,
            "unit": "ns/op",
            "extra": "931 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 2290,
            "unit": "B/op",
            "extra": "931 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "931 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine)",
            "value": 1297813,
            "unit": "ns/op\t    2291 B/op\t      17 allocs/op",
            "extra": "924 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1297813,
            "unit": "ns/op",
            "extra": "924 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 2291,
            "unit": "B/op",
            "extra": "924 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "924 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine)",
            "value": 1294115,
            "unit": "ns/op\t    2289 B/op\t      17 allocs/op",
            "extra": "930 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1294115,
            "unit": "ns/op",
            "extra": "930 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 2289,
            "unit": "B/op",
            "extra": "930 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "930 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine)",
            "value": 1293336,
            "unit": "ns/op\t    2290 B/op\t      17 allocs/op",
            "extra": "925 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1293336,
            "unit": "ns/op",
            "extra": "925 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 2290,
            "unit": "B/op",
            "extra": "925 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "925 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine)",
            "value": 1293514,
            "unit": "ns/op\t    2291 B/op\t      17 allocs/op",
            "extra": "924 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1293514,
            "unit": "ns/op",
            "extra": "924 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 2291,
            "unit": "B/op",
            "extra": "924 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "924 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine)",
            "value": 1299658,
            "unit": "ns/op\t    2292 B/op\t      17 allocs/op",
            "extra": "920 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1299658,
            "unit": "ns/op",
            "extra": "920 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 2292,
            "unit": "B/op",
            "extra": "920 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "920 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine)",
            "value": 1300098,
            "unit": "ns/op\t    2290 B/op\t      17 allocs/op",
            "extra": "928 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1300098,
            "unit": "ns/op",
            "extra": "928 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 2290,
            "unit": "B/op",
            "extra": "928 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "928 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine)",
            "value": 1299052,
            "unit": "ns/op\t    2290 B/op\t      17 allocs/op",
            "extra": "925 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1299052,
            "unit": "ns/op",
            "extra": "925 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 2290,
            "unit": "B/op",
            "extra": "925 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "925 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine)",
            "value": 1356471,
            "unit": "ns/op\t    2292 B/op\t      17 allocs/op",
            "extra": "922 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1356471,
            "unit": "ns/op",
            "extra": "922 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 2292,
            "unit": "B/op",
            "extra": "922 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "922 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine)",
            "value": 1295705,
            "unit": "ns/op\t    2291 B/op\t      17 allocs/op",
            "extra": "924 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1295705,
            "unit": "ns/op",
            "extra": "924 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 2291,
            "unit": "B/op",
            "extra": "924 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "924 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine)",
            "value": 1440144,
            "unit": "ns/op\t   30342 B/op\t      91 allocs/op",
            "extra": "852 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1440144,
            "unit": "ns/op",
            "extra": "852 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 30342,
            "unit": "B/op",
            "extra": "852 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 91,
            "unit": "allocs/op",
            "extra": "852 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine)",
            "value": 1443586,
            "unit": "ns/op\t   30253 B/op\t      91 allocs/op",
            "extra": "853 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1443586,
            "unit": "ns/op",
            "extra": "853 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 30253,
            "unit": "B/op",
            "extra": "853 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 91,
            "unit": "allocs/op",
            "extra": "853 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine)",
            "value": 1411598,
            "unit": "ns/op\t   30291 B/op\t      91 allocs/op",
            "extra": "861 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1411598,
            "unit": "ns/op",
            "extra": "861 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 30291,
            "unit": "B/op",
            "extra": "861 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 91,
            "unit": "allocs/op",
            "extra": "861 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine)",
            "value": 1427442,
            "unit": "ns/op\t   30205 B/op\t      91 allocs/op",
            "extra": "860 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1427442,
            "unit": "ns/op",
            "extra": "860 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 30205,
            "unit": "B/op",
            "extra": "860 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 91,
            "unit": "allocs/op",
            "extra": "860 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine)",
            "value": 1416219,
            "unit": "ns/op\t   30208 B/op\t      91 allocs/op",
            "extra": "860 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1416219,
            "unit": "ns/op",
            "extra": "860 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 30208,
            "unit": "B/op",
            "extra": "860 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 91,
            "unit": "allocs/op",
            "extra": "860 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine)",
            "value": 1417636,
            "unit": "ns/op\t   30279 B/op\t      91 allocs/op",
            "extra": "868 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1417636,
            "unit": "ns/op",
            "extra": "868 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 30279,
            "unit": "B/op",
            "extra": "868 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 91,
            "unit": "allocs/op",
            "extra": "868 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine)",
            "value": 1411592,
            "unit": "ns/op\t   30319 B/op\t      91 allocs/op",
            "extra": "867 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1411592,
            "unit": "ns/op",
            "extra": "867 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 30319,
            "unit": "B/op",
            "extra": "867 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 91,
            "unit": "allocs/op",
            "extra": "867 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine)",
            "value": 1418249,
            "unit": "ns/op\t   30250 B/op\t      91 allocs/op",
            "extra": "861 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1418249,
            "unit": "ns/op",
            "extra": "861 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 30250,
            "unit": "B/op",
            "extra": "861 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 91,
            "unit": "allocs/op",
            "extra": "861 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine)",
            "value": 1417512,
            "unit": "ns/op\t   30243 B/op\t      91 allocs/op",
            "extra": "861 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1417512,
            "unit": "ns/op",
            "extra": "861 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 30243,
            "unit": "B/op",
            "extra": "861 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 91,
            "unit": "allocs/op",
            "extra": "861 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine)",
            "value": 1412827,
            "unit": "ns/op\t   30249 B/op\t      91 allocs/op",
            "extra": "864 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1412827,
            "unit": "ns/op",
            "extra": "864 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 30249,
            "unit": "B/op",
            "extra": "864 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-32 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 91,
            "unit": "allocs/op",
            "extra": "864 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine)",
            "value": 1593052,
            "unit": "ns/op\t   91488 B/op\t     231 allocs/op",
            "extra": "794 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1593052,
            "unit": "ns/op",
            "extra": "794 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 91488,
            "unit": "B/op",
            "extra": "794 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 231,
            "unit": "allocs/op",
            "extra": "794 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine)",
            "value": 1593833,
            "unit": "ns/op\t   91095 B/op\t     231 allocs/op",
            "extra": "776 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1593833,
            "unit": "ns/op",
            "extra": "776 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 91095,
            "unit": "B/op",
            "extra": "776 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 231,
            "unit": "allocs/op",
            "extra": "776 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine)",
            "value": 1621513,
            "unit": "ns/op\t   91237 B/op\t     231 allocs/op",
            "extra": "788 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1621513,
            "unit": "ns/op",
            "extra": "788 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 91237,
            "unit": "B/op",
            "extra": "788 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 231,
            "unit": "allocs/op",
            "extra": "788 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine)",
            "value": 1598919,
            "unit": "ns/op\t   91233 B/op\t     231 allocs/op",
            "extra": "792 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1598919,
            "unit": "ns/op",
            "extra": "792 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 91233,
            "unit": "B/op",
            "extra": "792 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 231,
            "unit": "allocs/op",
            "extra": "792 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine)",
            "value": 1763589,
            "unit": "ns/op\t   91094 B/op\t     231 allocs/op",
            "extra": "782 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1763589,
            "unit": "ns/op",
            "extra": "782 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 91094,
            "unit": "B/op",
            "extra": "782 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 231,
            "unit": "allocs/op",
            "extra": "782 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine)",
            "value": 1622753,
            "unit": "ns/op\t   91152 B/op\t     231 allocs/op",
            "extra": "788 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1622753,
            "unit": "ns/op",
            "extra": "788 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 91152,
            "unit": "B/op",
            "extra": "788 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 231,
            "unit": "allocs/op",
            "extra": "788 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine)",
            "value": 1615467,
            "unit": "ns/op\t   91164 B/op\t     231 allocs/op",
            "extra": "783 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1615467,
            "unit": "ns/op",
            "extra": "783 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 91164,
            "unit": "B/op",
            "extra": "783 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 231,
            "unit": "allocs/op",
            "extra": "783 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine)",
            "value": 1621778,
            "unit": "ns/op\t   91154 B/op\t     231 allocs/op",
            "extra": "789 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1621778,
            "unit": "ns/op",
            "extra": "789 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 91154,
            "unit": "B/op",
            "extra": "789 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 231,
            "unit": "allocs/op",
            "extra": "789 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine)",
            "value": 1760749,
            "unit": "ns/op\t   91116 B/op\t     231 allocs/op",
            "extra": "752 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1760749,
            "unit": "ns/op",
            "extra": "752 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 91116,
            "unit": "B/op",
            "extra": "752 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 231,
            "unit": "allocs/op",
            "extra": "752 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine)",
            "value": 1638419,
            "unit": "ns/op\t   91370 B/op\t     231 allocs/op",
            "extra": "768 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1638419,
            "unit": "ns/op",
            "extra": "768 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 91370,
            "unit": "B/op",
            "extra": "768 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-100 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 231,
            "unit": "allocs/op",
            "extra": "768 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 9697982,
            "unit": "ns/op\t 1054578 B/op\t    2722 allocs/op",
            "extra": "586 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9697982,
            "unit": "ns/op",
            "extra": "586 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1054578,
            "unit": "B/op",
            "extra": "586 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 2722,
            "unit": "allocs/op",
            "extra": "586 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 8806216,
            "unit": "ns/op\t 1042831 B/op\t    2665 allocs/op",
            "extra": "532 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 8806216,
            "unit": "ns/op",
            "extra": "532 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1042831,
            "unit": "B/op",
            "extra": "532 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 2665,
            "unit": "allocs/op",
            "extra": "532 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 9263426,
            "unit": "ns/op\t 1041979 B/op\t    2672 allocs/op",
            "extra": "530 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9263426,
            "unit": "ns/op",
            "extra": "530 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1041979,
            "unit": "B/op",
            "extra": "530 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 2672,
            "unit": "allocs/op",
            "extra": "530 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 8948049,
            "unit": "ns/op\t 1041933 B/op\t    2692 allocs/op",
            "extra": "571 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 8948049,
            "unit": "ns/op",
            "extra": "571 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1041933,
            "unit": "B/op",
            "extra": "571 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 2692,
            "unit": "allocs/op",
            "extra": "571 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 9126695,
            "unit": "ns/op\t 1043272 B/op\t    2689 allocs/op",
            "extra": "566 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9126695,
            "unit": "ns/op",
            "extra": "566 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1043272,
            "unit": "B/op",
            "extra": "566 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 2689,
            "unit": "allocs/op",
            "extra": "566 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 8901055,
            "unit": "ns/op\t 1042865 B/op\t    2685 allocs/op",
            "extra": "559 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 8901055,
            "unit": "ns/op",
            "extra": "559 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1042865,
            "unit": "B/op",
            "extra": "559 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 2685,
            "unit": "allocs/op",
            "extra": "559 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 9086063,
            "unit": "ns/op\t 1043179 B/op\t    2686 allocs/op",
            "extra": "553 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9086063,
            "unit": "ns/op",
            "extra": "553 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1043179,
            "unit": "B/op",
            "extra": "553 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 2686,
            "unit": "allocs/op",
            "extra": "553 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 8880910,
            "unit": "ns/op\t 1044569 B/op\t    2676 allocs/op",
            "extra": "537 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 8880910,
            "unit": "ns/op",
            "extra": "537 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1044569,
            "unit": "B/op",
            "extra": "537 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 2676,
            "unit": "allocs/op",
            "extra": "537 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 9538613,
            "unit": "ns/op\t 1049282 B/op\t    2752 allocs/op",
            "extra": "600 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9538613,
            "unit": "ns/op",
            "extra": "600 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1049282,
            "unit": "B/op",
            "extra": "600 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 2752,
            "unit": "allocs/op",
            "extra": "600 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 9290325,
            "unit": "ns/op\t 1047404 B/op\t    2708 allocs/op",
            "extra": "582 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9290325,
            "unit": "ns/op",
            "extra": "582 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1047404,
            "unit": "B/op",
            "extra": "582 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWrite/batch-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 2708,
            "unit": "allocs/op",
            "extra": "582 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine)",
            "value": 355087,
            "unit": "ns/op\t    1393 B/op\t       8 allocs/op",
            "extra": "3644 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 355087,
            "unit": "ns/op",
            "extra": "3644 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1393,
            "unit": "B/op",
            "extra": "3644 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3644 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine)",
            "value": 328517,
            "unit": "ns/op\t    1388 B/op\t       8 allocs/op",
            "extra": "3638 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 328517,
            "unit": "ns/op",
            "extra": "3638 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1388,
            "unit": "B/op",
            "extra": "3638 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3638 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine)",
            "value": 327769,
            "unit": "ns/op\t    1388 B/op\t       8 allocs/op",
            "extra": "3558 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 327769,
            "unit": "ns/op",
            "extra": "3558 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1388,
            "unit": "B/op",
            "extra": "3558 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3558 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine)",
            "value": 327365,
            "unit": "ns/op\t    1388 B/op\t       8 allocs/op",
            "extra": "3608 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 327365,
            "unit": "ns/op",
            "extra": "3608 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1388,
            "unit": "B/op",
            "extra": "3608 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3608 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine)",
            "value": 326858,
            "unit": "ns/op\t    1385 B/op\t       8 allocs/op",
            "extra": "3668 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 326858,
            "unit": "ns/op",
            "extra": "3668 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1385,
            "unit": "B/op",
            "extra": "3668 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3668 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine)",
            "value": 328400,
            "unit": "ns/op\t    1393 B/op\t       8 allocs/op",
            "extra": "3626 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 328400,
            "unit": "ns/op",
            "extra": "3626 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1393,
            "unit": "B/op",
            "extra": "3626 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3626 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine)",
            "value": 328326,
            "unit": "ns/op\t    1393 B/op\t       8 allocs/op",
            "extra": "3619 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 328326,
            "unit": "ns/op",
            "extra": "3619 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1393,
            "unit": "B/op",
            "extra": "3619 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3619 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine)",
            "value": 329702,
            "unit": "ns/op\t    1386 B/op\t       8 allocs/op",
            "extra": "3636 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 329702,
            "unit": "ns/op",
            "extra": "3636 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1386,
            "unit": "B/op",
            "extra": "3636 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3636 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine)",
            "value": 328426,
            "unit": "ns/op\t    1386 B/op\t       8 allocs/op",
            "extra": "3663 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 328426,
            "unit": "ns/op",
            "extra": "3663 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1386,
            "unit": "B/op",
            "extra": "3663 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3663 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine)",
            "value": 328251,
            "unit": "ns/op\t    1387 B/op\t       8 allocs/op",
            "extra": "3616 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 328251,
            "unit": "ns/op",
            "extra": "3616 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1387,
            "unit": "B/op",
            "extra": "3616 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/single-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3616 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine)",
            "value": 327482,
            "unit": "ns/op\t    1753 B/op\t       8 allocs/op",
            "extra": "3616 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 327482,
            "unit": "ns/op",
            "extra": "3616 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1753,
            "unit": "B/op",
            "extra": "3616 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3616 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine)",
            "value": 327307,
            "unit": "ns/op\t    1746 B/op\t       8 allocs/op",
            "extra": "3669 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 327307,
            "unit": "ns/op",
            "extra": "3669 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1746,
            "unit": "B/op",
            "extra": "3669 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3669 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine)",
            "value": 328240,
            "unit": "ns/op\t    1749 B/op\t       8 allocs/op",
            "extra": "3610 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 328240,
            "unit": "ns/op",
            "extra": "3610 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1749,
            "unit": "B/op",
            "extra": "3610 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3610 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine)",
            "value": 328719,
            "unit": "ns/op\t    1744 B/op\t       8 allocs/op",
            "extra": "3634 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 328719,
            "unit": "ns/op",
            "extra": "3634 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1744,
            "unit": "B/op",
            "extra": "3634 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3634 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine)",
            "value": 328324,
            "unit": "ns/op\t    1747 B/op\t       8 allocs/op",
            "extra": "3654 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 328324,
            "unit": "ns/op",
            "extra": "3654 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1747,
            "unit": "B/op",
            "extra": "3654 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3654 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine)",
            "value": 326587,
            "unit": "ns/op\t    1748 B/op\t       8 allocs/op",
            "extra": "3643 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 326587,
            "unit": "ns/op",
            "extra": "3643 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1748,
            "unit": "B/op",
            "extra": "3643 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3643 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine)",
            "value": 329076,
            "unit": "ns/op\t    1747 B/op\t       8 allocs/op",
            "extra": "3650 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 329076,
            "unit": "ns/op",
            "extra": "3650 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1747,
            "unit": "B/op",
            "extra": "3650 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3650 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine)",
            "value": 328531,
            "unit": "ns/op\t    1748 B/op\t       8 allocs/op",
            "extra": "3640 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 328531,
            "unit": "ns/op",
            "extra": "3640 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1748,
            "unit": "B/op",
            "extra": "3640 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3640 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine)",
            "value": 328621,
            "unit": "ns/op\t    1748 B/op\t       8 allocs/op",
            "extra": "3660 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 328621,
            "unit": "ns/op",
            "extra": "3660 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1748,
            "unit": "B/op",
            "extra": "3660 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3660 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine)",
            "value": 328510,
            "unit": "ns/op\t    1756 B/op\t       8 allocs/op",
            "extra": "3595 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 328510,
            "unit": "ns/op",
            "extra": "3595 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1756,
            "unit": "B/op",
            "extra": "3595 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/multi-room (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "3595 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 44969,
            "unit": "ns/op\t    1263 B/op\t       4 allocs/op",
            "extra": "27111 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 44969,
            "unit": "ns/op",
            "extra": "27111 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1263,
            "unit": "B/op",
            "extra": "27111 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "27111 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 45209,
            "unit": "ns/op\t    1257 B/op\t       4 allocs/op",
            "extra": "27098 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 45209,
            "unit": "ns/op",
            "extra": "27098 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1257,
            "unit": "B/op",
            "extra": "27098 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "27098 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 44972,
            "unit": "ns/op\t    1255 B/op\t       4 allocs/op",
            "extra": "27039 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 44972,
            "unit": "ns/op",
            "extra": "27039 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1255,
            "unit": "B/op",
            "extra": "27039 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "27039 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 45102,
            "unit": "ns/op\t    1254 B/op\t       4 allocs/op",
            "extra": "26857 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 45102,
            "unit": "ns/op",
            "extra": "26857 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1254,
            "unit": "B/op",
            "extra": "26857 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "26857 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 45324,
            "unit": "ns/op\t    1255 B/op\t       4 allocs/op",
            "extra": "27030 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 45324,
            "unit": "ns/op",
            "extra": "27030 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1255,
            "unit": "B/op",
            "extra": "27030 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "27030 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 45272,
            "unit": "ns/op\t    1253 B/op\t       4 allocs/op",
            "extra": "26998 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 45272,
            "unit": "ns/op",
            "extra": "26998 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1253,
            "unit": "B/op",
            "extra": "26998 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "26998 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 45214,
            "unit": "ns/op\t    1257 B/op\t       4 allocs/op",
            "extra": "26860 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 45214,
            "unit": "ns/op",
            "extra": "26860 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1257,
            "unit": "B/op",
            "extra": "26860 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "26860 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 45003,
            "unit": "ns/op\t    1254 B/op\t       4 allocs/op",
            "extra": "26929 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 45003,
            "unit": "ns/op",
            "extra": "26929 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1254,
            "unit": "B/op",
            "extra": "26929 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "26929 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 45024,
            "unit": "ns/op\t    1254 B/op\t       4 allocs/op",
            "extra": "26947 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 45024,
            "unit": "ns/op",
            "extra": "26947 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1254,
            "unit": "B/op",
            "extra": "26947 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "26947 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 45089,
            "unit": "ns/op\t    1254 B/op\t       4 allocs/op",
            "extra": "27093 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 45089,
            "unit": "ns/op",
            "extra": "27093 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1254,
            "unit": "B/op",
            "extra": "27093 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-128-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "27093 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 13533,
            "unit": "ns/op\t    1188 B/op\t       3 allocs/op",
            "extra": "89920 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 13533,
            "unit": "ns/op",
            "extra": "89920 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1188,
            "unit": "B/op",
            "extra": "89920 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "89920 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 13427,
            "unit": "ns/op\t    1180 B/op\t       3 allocs/op",
            "extra": "89362 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 13427,
            "unit": "ns/op",
            "extra": "89362 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1180,
            "unit": "B/op",
            "extra": "89362 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "89362 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 13650,
            "unit": "ns/op\t    1181 B/op\t       3 allocs/op",
            "extra": "88113 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 13650,
            "unit": "ns/op",
            "extra": "88113 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1181,
            "unit": "B/op",
            "extra": "88113 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "88113 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 13466,
            "unit": "ns/op\t    1179 B/op\t       3 allocs/op",
            "extra": "88389 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 13466,
            "unit": "ns/op",
            "extra": "88389 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1179,
            "unit": "B/op",
            "extra": "88389 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "88389 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 13498,
            "unit": "ns/op\t    1185 B/op\t       3 allocs/op",
            "extra": "85695 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 13498,
            "unit": "ns/op",
            "extra": "85695 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1185,
            "unit": "B/op",
            "extra": "85695 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "85695 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 13449,
            "unit": "ns/op\t    1176 B/op\t       3 allocs/op",
            "extra": "88537 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 13449,
            "unit": "ns/op",
            "extra": "88537 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1176,
            "unit": "B/op",
            "extra": "88537 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "88537 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 13587,
            "unit": "ns/op\t    1177 B/op\t       3 allocs/op",
            "extra": "89266 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 13587,
            "unit": "ns/op",
            "extra": "89266 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1177,
            "unit": "B/op",
            "extra": "89266 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "89266 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 13451,
            "unit": "ns/op\t    1183 B/op\t       3 allocs/op",
            "extra": "87655 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 13451,
            "unit": "ns/op",
            "extra": "87655 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1183,
            "unit": "B/op",
            "extra": "87655 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "87655 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 13483,
            "unit": "ns/op\t    1179 B/op\t       3 allocs/op",
            "extra": "88258 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 13483,
            "unit": "ns/op",
            "extra": "88258 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1179,
            "unit": "B/op",
            "extra": "88258 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "88258 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine)",
            "value": 13374,
            "unit": "ns/op\t    1183 B/op\t       3 allocs/op",
            "extra": "86676 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 13374,
            "unit": "ns/op",
            "extra": "86676 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 1183,
            "unit": "B/op",
            "extra": "86676 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineMessageWriteParallel/saturated-512-producers (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "86676 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine)",
            "value": 666.4,
            "unit": "ns/op\t    3129 B/op\t       4 allocs/op",
            "extra": "1792020 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 666.4,
            "unit": "ns/op",
            "extra": "1792020 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3129,
            "unit": "B/op",
            "extra": "1792020 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1792020 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine)",
            "value": 743.6,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1591158 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 743.6,
            "unit": "ns/op",
            "extra": "1591158 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1591158 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1591158 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine)",
            "value": 718.5,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1664149 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 718.5,
            "unit": "ns/op",
            "extra": "1664149 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1664149 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1664149 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine)",
            "value": 745.4,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1616296 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 745.4,
            "unit": "ns/op",
            "extra": "1616296 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1616296 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1616296 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine)",
            "value": 735.9,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1654633 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 735.9,
            "unit": "ns/op",
            "extra": "1654633 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1654633 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1654633 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine)",
            "value": 721.5,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1652744 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 721.5,
            "unit": "ns/op",
            "extra": "1652744 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1652744 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1652744 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine)",
            "value": 732.6,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1635274 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 732.6,
            "unit": "ns/op",
            "extra": "1635274 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1635274 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1635274 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine)",
            "value": 725.3,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1654598 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 725.3,
            "unit": "ns/op",
            "extra": "1654598 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1654598 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1654598 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine)",
            "value": 740.8,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1607462 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 740.8,
            "unit": "ns/op",
            "extra": "1607462 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1607462 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1607462 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine)",
            "value": 733.5,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1637583 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 733.5,
            "unit": "ns/op",
            "extra": "1637583 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1637583 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/newest-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1637583 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine)",
            "value": 3175,
            "unit": "ns/op\t    8265 B/op\t      32 allocs/op",
            "extra": "335737 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 3175,
            "unit": "ns/op",
            "extra": "335737 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8265,
            "unit": "B/op",
            "extra": "335737 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 32,
            "unit": "allocs/op",
            "extra": "335737 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine)",
            "value": 3556,
            "unit": "ns/op\t    8264 B/op\t      32 allocs/op",
            "extra": "288092 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 3556,
            "unit": "ns/op",
            "extra": "288092 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8264,
            "unit": "B/op",
            "extra": "288092 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 32,
            "unit": "allocs/op",
            "extra": "288092 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine)",
            "value": 3388,
            "unit": "ns/op\t    8265 B/op\t      32 allocs/op",
            "extra": "307106 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 3388,
            "unit": "ns/op",
            "extra": "307106 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8265,
            "unit": "B/op",
            "extra": "307106 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 32,
            "unit": "allocs/op",
            "extra": "307106 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine)",
            "value": 3411,
            "unit": "ns/op\t    8264 B/op\t      32 allocs/op",
            "extra": "329325 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 3411,
            "unit": "ns/op",
            "extra": "329325 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8264,
            "unit": "B/op",
            "extra": "329325 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 32,
            "unit": "allocs/op",
            "extra": "329325 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine)",
            "value": 3519,
            "unit": "ns/op\t    8265 B/op\t      32 allocs/op",
            "extra": "339008 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 3519,
            "unit": "ns/op",
            "extra": "339008 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8265,
            "unit": "B/op",
            "extra": "339008 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 32,
            "unit": "allocs/op",
            "extra": "339008 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine)",
            "value": 3455,
            "unit": "ns/op\t    8265 B/op\t      32 allocs/op",
            "extra": "330453 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 3455,
            "unit": "ns/op",
            "extra": "330453 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8265,
            "unit": "B/op",
            "extra": "330453 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 32,
            "unit": "allocs/op",
            "extra": "330453 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine)",
            "value": 3621,
            "unit": "ns/op\t    8265 B/op\t      32 allocs/op",
            "extra": "320350 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 3621,
            "unit": "ns/op",
            "extra": "320350 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8265,
            "unit": "B/op",
            "extra": "320350 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 32,
            "unit": "allocs/op",
            "extra": "320350 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine)",
            "value": 3461,
            "unit": "ns/op\t    8265 B/op\t      32 allocs/op",
            "extra": "317761 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 3461,
            "unit": "ns/op",
            "extra": "317761 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8265,
            "unit": "B/op",
            "extra": "317761 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 32,
            "unit": "allocs/op",
            "extra": "317761 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine)",
            "value": 3448,
            "unit": "ns/op\t    8265 B/op\t      32 allocs/op",
            "extra": "325336 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 3448,
            "unit": "ns/op",
            "extra": "325336 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8265,
            "unit": "B/op",
            "extra": "325336 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 32,
            "unit": "allocs/op",
            "extra": "325336 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine)",
            "value": 3505,
            "unit": "ns/op\t    8265 B/op\t      32 allocs/op",
            "extra": "291828 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 3505,
            "unit": "ns/op",
            "extra": "291828 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8265,
            "unit": "B/op",
            "extra": "291828 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/deep-page (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 32,
            "unit": "allocs/op",
            "extra": "291828 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine)",
            "value": 671.9,
            "unit": "ns/op\t    3129 B/op\t       4 allocs/op",
            "extra": "1735582 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 671.9,
            "unit": "ns/op",
            "extra": "1735582 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3129,
            "unit": "B/op",
            "extra": "1735582 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1735582 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine)",
            "value": 733.7,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1623187 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 733.7,
            "unit": "ns/op",
            "extra": "1623187 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1623187 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1623187 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine)",
            "value": 732.8,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1625536 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 732.8,
            "unit": "ns/op",
            "extra": "1625536 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1625536 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1625536 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine)",
            "value": 721.7,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1646659 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 721.7,
            "unit": "ns/op",
            "extra": "1646659 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1646659 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1646659 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine)",
            "value": 735.1,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1613828 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 735.1,
            "unit": "ns/op",
            "extra": "1613828 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1613828 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1613828 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine)",
            "value": 730.2,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1643709 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 730.2,
            "unit": "ns/op",
            "extra": "1643709 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1643709 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1643709 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine)",
            "value": 730.2,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1639275 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 730.2,
            "unit": "ns/op",
            "extra": "1639275 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1639275 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1639275 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine)",
            "value": 724.7,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1647639 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 724.7,
            "unit": "ns/op",
            "extra": "1647639 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1647639 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1647639 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine)",
            "value": 742.6,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1593715 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 742.6,
            "unit": "ns/op",
            "extra": "1593715 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1593715 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1593715 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine)",
            "value": 724.9,
            "unit": "ns/op\t    3128 B/op\t       4 allocs/op",
            "extra": "1647194 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 724.9,
            "unit": "ns/op",
            "extra": "1647194 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3128,
            "unit": "B/op",
            "extra": "1647194 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRead/sync-pull-after (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "1647194 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine)",
            "value": 2863,
            "unit": "ns/op\t    3929 B/op\t      28 allocs/op",
            "extra": "394606 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2863,
            "unit": "ns/op",
            "extra": "394606 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3929,
            "unit": "B/op",
            "extra": "394606 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "394606 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine)",
            "value": 2948,
            "unit": "ns/op\t    3928 B/op\t      28 allocs/op",
            "extra": "409990 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2948,
            "unit": "ns/op",
            "extra": "409990 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3928,
            "unit": "B/op",
            "extra": "409990 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "409990 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine)",
            "value": 2942,
            "unit": "ns/op\t    3928 B/op\t      28 allocs/op",
            "extra": "359590 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2942,
            "unit": "ns/op",
            "extra": "359590 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3928,
            "unit": "B/op",
            "extra": "359590 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "359590 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine)",
            "value": 2932,
            "unit": "ns/op\t    3929 B/op\t      28 allocs/op",
            "extra": "367776 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2932,
            "unit": "ns/op",
            "extra": "367776 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3929,
            "unit": "B/op",
            "extra": "367776 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "367776 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine)",
            "value": 2959,
            "unit": "ns/op\t    3929 B/op\t      28 allocs/op",
            "extra": "396154 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2959,
            "unit": "ns/op",
            "extra": "396154 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3929,
            "unit": "B/op",
            "extra": "396154 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "396154 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine)",
            "value": 2990,
            "unit": "ns/op\t    3929 B/op\t      28 allocs/op",
            "extra": "371949 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2990,
            "unit": "ns/op",
            "extra": "371949 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3929,
            "unit": "B/op",
            "extra": "371949 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "371949 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine)",
            "value": 2930,
            "unit": "ns/op\t    3928 B/op\t      28 allocs/op",
            "extra": "366547 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2930,
            "unit": "ns/op",
            "extra": "366547 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3928,
            "unit": "B/op",
            "extra": "366547 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "366547 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine)",
            "value": 2975,
            "unit": "ns/op\t    3928 B/op\t      28 allocs/op",
            "extra": "358208 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2975,
            "unit": "ns/op",
            "extra": "358208 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3928,
            "unit": "B/op",
            "extra": "358208 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "358208 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine)",
            "value": 2964,
            "unit": "ns/op\t    3929 B/op\t      28 allocs/op",
            "extra": "376274 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2964,
            "unit": "ns/op",
            "extra": "376274 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3929,
            "unit": "B/op",
            "extra": "376274 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "376274 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine)",
            "value": 2992,
            "unit": "ns/op\t    3928 B/op\t      28 allocs/op",
            "extra": "376938 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2992,
            "unit": "ns/op",
            "extra": "376938 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3928,
            "unit": "B/op",
            "extra": "376938 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/recent-200 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "376938 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine)",
            "value": 371143,
            "unit": "ns/op\t    3823 B/op\t      28 allocs/op",
            "extra": "3142 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 371143,
            "unit": "ns/op",
            "extra": "3142 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3823,
            "unit": "B/op",
            "extra": "3142 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "3142 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine)",
            "value": 366395,
            "unit": "ns/op\t    3823 B/op\t      28 allocs/op",
            "extra": "3246 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 366395,
            "unit": "ns/op",
            "extra": "3246 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3823,
            "unit": "B/op",
            "extra": "3246 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "3246 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine)",
            "value": 373138,
            "unit": "ns/op\t    3823 B/op\t      28 allocs/op",
            "extra": "3198 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 373138,
            "unit": "ns/op",
            "extra": "3198 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3823,
            "unit": "B/op",
            "extra": "3198 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "3198 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine)",
            "value": 369640,
            "unit": "ns/op\t    3823 B/op\t      28 allocs/op",
            "extra": "3228 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 369640,
            "unit": "ns/op",
            "extra": "3228 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3823,
            "unit": "B/op",
            "extra": "3228 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "3228 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine)",
            "value": 427926,
            "unit": "ns/op\t    3824 B/op\t      28 allocs/op",
            "extra": "2914 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 427926,
            "unit": "ns/op",
            "extra": "2914 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3824,
            "unit": "B/op",
            "extra": "2914 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "2914 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine)",
            "value": 369273,
            "unit": "ns/op\t    3823 B/op\t      28 allocs/op",
            "extra": "3200 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 369273,
            "unit": "ns/op",
            "extra": "3200 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3823,
            "unit": "B/op",
            "extra": "3200 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "3200 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine)",
            "value": 382023,
            "unit": "ns/op\t    3823 B/op\t      28 allocs/op",
            "extra": "3132 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 382023,
            "unit": "ns/op",
            "extra": "3132 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3823,
            "unit": "B/op",
            "extra": "3132 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "3132 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine)",
            "value": 376527,
            "unit": "ns/op\t    3823 B/op\t      28 allocs/op",
            "extra": "3144 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 376527,
            "unit": "ns/op",
            "extra": "3144 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3823,
            "unit": "B/op",
            "extra": "3144 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "3144 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine)",
            "value": 374684,
            "unit": "ns/op\t    3823 B/op\t      28 allocs/op",
            "extra": "3216 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 374684,
            "unit": "ns/op",
            "extra": "3216 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3823,
            "unit": "B/op",
            "extra": "3216 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "3216 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine)",
            "value": 381951,
            "unit": "ns/op\t    3823 B/op\t      28 allocs/op",
            "extra": "3163 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 381951,
            "unit": "ns/op",
            "extra": "3163 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3823,
            "unit": "B/op",
            "extra": "3163 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineSearch/deep-scan-5000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 28,
            "unit": "allocs/op",
            "extra": "3163 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine)",
            "value": 1519106,
            "unit": "ns/op\t    3194 B/op\t      31 allocs/op",
            "extra": "783 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1519106,
            "unit": "ns/op",
            "extra": "783 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3194,
            "unit": "B/op",
            "extra": "783 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 31,
            "unit": "allocs/op",
            "extra": "783 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine)",
            "value": 1512389,
            "unit": "ns/op\t    3191 B/op\t      31 allocs/op",
            "extra": "792 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1512389,
            "unit": "ns/op",
            "extra": "792 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3191,
            "unit": "B/op",
            "extra": "792 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 31,
            "unit": "allocs/op",
            "extra": "792 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine)",
            "value": 1498019,
            "unit": "ns/op\t    3188 B/op\t      31 allocs/op",
            "extra": "800 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1498019,
            "unit": "ns/op",
            "extra": "800 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3188,
            "unit": "B/op",
            "extra": "800 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 31,
            "unit": "allocs/op",
            "extra": "800 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine)",
            "value": 1499297,
            "unit": "ns/op\t    3190 B/op\t      31 allocs/op",
            "extra": "796 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1499297,
            "unit": "ns/op",
            "extra": "796 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3190,
            "unit": "B/op",
            "extra": "796 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 31,
            "unit": "allocs/op",
            "extra": "796 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine)",
            "value": 1501605,
            "unit": "ns/op\t    3190 B/op\t      31 allocs/op",
            "extra": "794 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1501605,
            "unit": "ns/op",
            "extra": "794 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3190,
            "unit": "B/op",
            "extra": "794 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 31,
            "unit": "allocs/op",
            "extra": "794 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine)",
            "value": 1490283,
            "unit": "ns/op\t    3186 B/op\t      31 allocs/op",
            "extra": "805 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1490283,
            "unit": "ns/op",
            "extra": "805 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3186,
            "unit": "B/op",
            "extra": "805 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 31,
            "unit": "allocs/op",
            "extra": "805 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine)",
            "value": 1498464,
            "unit": "ns/op\t    3193 B/op\t      31 allocs/op",
            "extra": "787 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1498464,
            "unit": "ns/op",
            "extra": "787 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3193,
            "unit": "B/op",
            "extra": "787 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 31,
            "unit": "allocs/op",
            "extra": "787 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine)",
            "value": 1496535,
            "unit": "ns/op\t    3189 B/op\t      31 allocs/op",
            "extra": "796 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1496535,
            "unit": "ns/op",
            "extra": "796 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3189,
            "unit": "B/op",
            "extra": "796 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 31,
            "unit": "allocs/op",
            "extra": "796 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine)",
            "value": 1497888,
            "unit": "ns/op\t    3188 B/op\t      31 allocs/op",
            "extra": "800 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1497888,
            "unit": "ns/op",
            "extra": "800 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3188,
            "unit": "B/op",
            "extra": "800 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 31,
            "unit": "allocs/op",
            "extra": "800 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine)",
            "value": 1507248,
            "unit": "ns/op\t    3198 B/op\t      31 allocs/op",
            "extra": "789 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1507248,
            "unit": "ns/op",
            "extra": "789 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3198,
            "unit": "B/op",
            "extra": "789 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineRecall/write-and-recall (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 31,
            "unit": "allocs/op",
            "extra": "789 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 9.259,
            "unit": "ns/op\t       8 B/op\t       0 allocs/op",
            "extra": "129012528 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9.259,
            "unit": "ns/op",
            "extra": "129012528 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8,
            "unit": "B/op",
            "extra": "129012528 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "129012528 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 9.104,
            "unit": "ns/op\t       8 B/op\t       0 allocs/op",
            "extra": "132088012 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9.104,
            "unit": "ns/op",
            "extra": "132088012 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8,
            "unit": "B/op",
            "extra": "132088012 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "132088012 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 9.171,
            "unit": "ns/op\t       8 B/op\t       0 allocs/op",
            "extra": "131146605 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9.171,
            "unit": "ns/op",
            "extra": "131146605 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8,
            "unit": "B/op",
            "extra": "131146605 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "131146605 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 9.189,
            "unit": "ns/op\t       8 B/op\t       0 allocs/op",
            "extra": "131218316 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9.189,
            "unit": "ns/op",
            "extra": "131218316 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8,
            "unit": "B/op",
            "extra": "131218316 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "131218316 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 9.321,
            "unit": "ns/op\t       8 B/op\t       0 allocs/op",
            "extra": "127998789 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9.321,
            "unit": "ns/op",
            "extra": "127998789 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8,
            "unit": "B/op",
            "extra": "127998789 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "127998789 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 9.188,
            "unit": "ns/op\t       8 B/op\t       1 allocs/op",
            "extra": "130393878 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9.188,
            "unit": "ns/op",
            "extra": "130393878 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8,
            "unit": "B/op",
            "extra": "130393878 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "130393878 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 9.19,
            "unit": "ns/op\t       8 B/op\t       1 allocs/op",
            "extra": "130559958 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9.19,
            "unit": "ns/op",
            "extra": "130559958 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8,
            "unit": "B/op",
            "extra": "130559958 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "130559958 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 9.132,
            "unit": "ns/op\t       8 B/op\t       1 allocs/op",
            "extra": "131840438 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9.132,
            "unit": "ns/op",
            "extra": "131840438 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8,
            "unit": "B/op",
            "extra": "131840438 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "131840438 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 9.294,
            "unit": "ns/op\t       8 B/op\t       1 allocs/op",
            "extra": "128321132 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9.294,
            "unit": "ns/op",
            "extra": "128321132 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8,
            "unit": "B/op",
            "extra": "128321132 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "128321132 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine)",
            "value": 9.189,
            "unit": "ns/op\t       8 B/op\t       1 allocs/op",
            "extra": "130696800 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 9.189,
            "unit": "ns/op",
            "extra": "130696800 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 8,
            "unit": "B/op",
            "extra": "130696800 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/encode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "130696800 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 0.8896,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 0.8896,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 0.6745,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 0.6745,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 0.8896,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 0.8896,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 0.8932,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 0.8932,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 0.8995,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 0.8995,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 0.667,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 0.667,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 0.667,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 0.667,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 0.9421,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 0.9421,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 0.89,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 0.89,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine)",
            "value": 0.8955,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 0.8955,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRoomHead/decode (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine)",
            "value": 2955,
            "unit": "ns/op\t    3681 B/op\t      22 allocs/op",
            "extra": "381381 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2955,
            "unit": "ns/op",
            "extra": "381381 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3681,
            "unit": "B/op",
            "extra": "381381 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "381381 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine)",
            "value": 2938,
            "unit": "ns/op\t    3680 B/op\t      22 allocs/op",
            "extra": "408656 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2938,
            "unit": "ns/op",
            "extra": "408656 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3680,
            "unit": "B/op",
            "extra": "408656 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "408656 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine)",
            "value": 2861,
            "unit": "ns/op\t    3680 B/op\t      22 allocs/op",
            "extra": "384003 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2861,
            "unit": "ns/op",
            "extra": "384003 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3680,
            "unit": "B/op",
            "extra": "384003 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "384003 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine)",
            "value": 2855,
            "unit": "ns/op\t    3680 B/op\t      22 allocs/op",
            "extra": "419188 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2855,
            "unit": "ns/op",
            "extra": "419188 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3680,
            "unit": "B/op",
            "extra": "419188 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "419188 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine)",
            "value": 2871,
            "unit": "ns/op\t    3680 B/op\t      22 allocs/op",
            "extra": "359586 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2871,
            "unit": "ns/op",
            "extra": "359586 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3680,
            "unit": "B/op",
            "extra": "359586 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "359586 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine)",
            "value": 2995,
            "unit": "ns/op\t    3680 B/op\t      22 allocs/op",
            "extra": "350197 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2995,
            "unit": "ns/op",
            "extra": "350197 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3680,
            "unit": "B/op",
            "extra": "350197 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "350197 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine)",
            "value": 2950,
            "unit": "ns/op\t    3680 B/op\t      22 allocs/op",
            "extra": "384824 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2950,
            "unit": "ns/op",
            "extra": "384824 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3680,
            "unit": "B/op",
            "extra": "384824 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "384824 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine)",
            "value": 2895,
            "unit": "ns/op\t    3680 B/op\t      22 allocs/op",
            "extra": "368223 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2895,
            "unit": "ns/op",
            "extra": "368223 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3680,
            "unit": "B/op",
            "extra": "368223 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "368223 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine)",
            "value": 2909,
            "unit": "ns/op\t    3680 B/op\t      22 allocs/op",
            "extra": "367034 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2909,
            "unit": "ns/op",
            "extra": "367034 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3680,
            "unit": "B/op",
            "extra": "367034 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "367034 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine)",
            "value": 2870,
            "unit": "ns/op\t    3680 B/op\t      22 allocs/op",
            "extra": "378528 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 2870,
            "unit": "ns/op",
            "extra": "378528 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 3680,
            "unit": "B/op",
            "extra": "378528 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-on-old-messages (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "378528 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine)",
            "value": 11968,
            "unit": "ns/op\t    6849 B/op\t     166 allocs/op",
            "extra": "100692 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 11968,
            "unit": "ns/op",
            "extra": "100692 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 6849,
            "unit": "B/op",
            "extra": "100692 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 166,
            "unit": "allocs/op",
            "extra": "100692 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine)",
            "value": 13285,
            "unit": "ns/op\t    6848 B/op\t     166 allocs/op",
            "extra": "86803 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 13285,
            "unit": "ns/op",
            "extra": "86803 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 6848,
            "unit": "B/op",
            "extra": "86803 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 166,
            "unit": "allocs/op",
            "extra": "86803 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine)",
            "value": 12314,
            "unit": "ns/op\t    6848 B/op\t     166 allocs/op",
            "extra": "94399 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 12314,
            "unit": "ns/op",
            "extra": "94399 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 6848,
            "unit": "B/op",
            "extra": "94399 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 166,
            "unit": "allocs/op",
            "extra": "94399 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine)",
            "value": 11974,
            "unit": "ns/op\t    6848 B/op\t     166 allocs/op",
            "extra": "95230 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 11974,
            "unit": "ns/op",
            "extra": "95230 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 6848,
            "unit": "B/op",
            "extra": "95230 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 166,
            "unit": "allocs/op",
            "extra": "95230 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine)",
            "value": 12381,
            "unit": "ns/op\t    6849 B/op\t     166 allocs/op",
            "extra": "94869 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 12381,
            "unit": "ns/op",
            "extra": "94869 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 6849,
            "unit": "B/op",
            "extra": "94869 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 166,
            "unit": "allocs/op",
            "extra": "94869 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine)",
            "value": 12455,
            "unit": "ns/op\t    6848 B/op\t     166 allocs/op",
            "extra": "92790 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 12455,
            "unit": "ns/op",
            "extra": "92790 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 6848,
            "unit": "B/op",
            "extra": "92790 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 166,
            "unit": "allocs/op",
            "extra": "92790 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine)",
            "value": 12836,
            "unit": "ns/op\t    6848 B/op\t     166 allocs/op",
            "extra": "90980 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 12836,
            "unit": "ns/op",
            "extra": "90980 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 6848,
            "unit": "B/op",
            "extra": "90980 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 166,
            "unit": "allocs/op",
            "extra": "90980 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine)",
            "value": 12276,
            "unit": "ns/op\t    6849 B/op\t     166 allocs/op",
            "extra": "92524 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 12276,
            "unit": "ns/op",
            "extra": "92524 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 6849,
            "unit": "B/op",
            "extra": "92524 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 166,
            "unit": "allocs/op",
            "extra": "92524 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine)",
            "value": 12109,
            "unit": "ns/op\t    6848 B/op\t     166 allocs/op",
            "extra": "97755 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 12109,
            "unit": "ns/op",
            "extra": "97755 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 6848,
            "unit": "B/op",
            "extra": "97755 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 166,
            "unit": "allocs/op",
            "extra": "97755 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine)",
            "value": 12836,
            "unit": "ns/op\t    6848 B/op\t     166 allocs/op",
            "extra": "90889 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 12836,
            "unit": "ns/op",
            "extra": "90889 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 6848,
            "unit": "B/op",
            "extra": "90889 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadWithReactions/reactions-everywhere (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 166,
            "unit": "allocs/op",
            "extra": "90889 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine)",
            "value": 1104,
            "unit": "ns/op\t      81 B/op\t       3 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1104,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 81,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine)",
            "value": 1192,
            "unit": "ns/op\t      81 B/op\t       3 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1192,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 81,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine)",
            "value": 1088,
            "unit": "ns/op\t      81 B/op\t       3 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1088,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 81,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine)",
            "value": 1110,
            "unit": "ns/op\t      81 B/op\t       3 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1110,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 81,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine)",
            "value": 1102,
            "unit": "ns/op\t      81 B/op\t       3 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1102,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 81,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine)",
            "value": 1077,
            "unit": "ns/op\t      81 B/op\t       3 allocs/op",
            "extra": "993807 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1077,
            "unit": "ns/op",
            "extra": "993807 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 81,
            "unit": "B/op",
            "extra": "993807 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "993807 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine)",
            "value": 1092,
            "unit": "ns/op\t      81 B/op\t       3 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1092,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 81,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine)",
            "value": 1116,
            "unit": "ns/op\t      81 B/op\t       3 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1116,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 81,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine)",
            "value": 1117,
            "unit": "ns/op\t      81 B/op\t       3 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1117,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 81,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine)",
            "value": 1102,
            "unit": "ns/op\t      81 B/op\t       3 allocs/op",
            "extra": "977341 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 1102,
            "unit": "ns/op",
            "extra": "977341 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 81,
            "unit": "B/op",
            "extra": "977341 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/mark-read (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "977341 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 203.3,
            "unit": "ns/op\t      80 B/op\t       3 allocs/op",
            "extra": "5823817 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 203.3,
            "unit": "ns/op",
            "extra": "5823817 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "5823817 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5823817 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 207.4,
            "unit": "ns/op\t      80 B/op\t       3 allocs/op",
            "extra": "5699210 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 207.4,
            "unit": "ns/op",
            "extra": "5699210 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "5699210 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5699210 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 204.6,
            "unit": "ns/op\t      80 B/op\t       3 allocs/op",
            "extra": "5780348 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 204.6,
            "unit": "ns/op",
            "extra": "5780348 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "5780348 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5780348 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 208,
            "unit": "ns/op\t      80 B/op\t       3 allocs/op",
            "extra": "5756850 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 208,
            "unit": "ns/op",
            "extra": "5756850 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "5756850 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5756850 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 206.6,
            "unit": "ns/op\t      80 B/op\t       3 allocs/op",
            "extra": "5843886 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 206.6,
            "unit": "ns/op",
            "extra": "5843886 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "5843886 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5843886 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 225.3,
            "unit": "ns/op\t      80 B/op\t       3 allocs/op",
            "extra": "5356621 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 225.3,
            "unit": "ns/op",
            "extra": "5356621 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "5356621 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5356621 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 211,
            "unit": "ns/op\t      80 B/op\t       3 allocs/op",
            "extra": "5630587 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 211,
            "unit": "ns/op",
            "extra": "5630587 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "5630587 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5630587 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 206,
            "unit": "ns/op\t      80 B/op\t       3 allocs/op",
            "extra": "5723952 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 206,
            "unit": "ns/op",
            "extra": "5723952 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "5723952 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5723952 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 203,
            "unit": "ns/op\t      80 B/op\t       3 allocs/op",
            "extra": "5872291 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 203,
            "unit": "ns/op",
            "extra": "5872291 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "5872291 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5872291 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine)",
            "value": 205.6,
            "unit": "ns/op\t      80 B/op\t       3 allocs/op",
            "extra": "5783746 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 205.6,
            "unit": "ns/op",
            "extra": "5783746 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "5783746 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-seq (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5783746 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 51090,
            "unit": "ns/op\t   71063 B/op\t      10 allocs/op",
            "extra": "23461 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 51090,
            "unit": "ns/op",
            "extra": "23461 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 71063,
            "unit": "B/op",
            "extra": "23461 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "23461 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 51708,
            "unit": "ns/op\t   71059 B/op\t      10 allocs/op",
            "extra": "23227 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 51708,
            "unit": "ns/op",
            "extra": "23227 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 71059,
            "unit": "B/op",
            "extra": "23227 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "23227 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 51492,
            "unit": "ns/op\t   71059 B/op\t      10 allocs/op",
            "extra": "23430 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 51492,
            "unit": "ns/op",
            "extra": "23430 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 71059,
            "unit": "B/op",
            "extra": "23430 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "23430 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 52210,
            "unit": "ns/op\t   71060 B/op\t      10 allocs/op",
            "extra": "23251 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 52210,
            "unit": "ns/op",
            "extra": "23251 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 71060,
            "unit": "B/op",
            "extra": "23251 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "23251 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 55747,
            "unit": "ns/op\t   71060 B/op\t      10 allocs/op",
            "extra": "19984 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 55747,
            "unit": "ns/op",
            "extra": "19984 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 71060,
            "unit": "B/op",
            "extra": "19984 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "19984 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 51280,
            "unit": "ns/op\t   71060 B/op\t      10 allocs/op",
            "extra": "23384 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 51280,
            "unit": "ns/op",
            "extra": "23384 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 71060,
            "unit": "B/op",
            "extra": "23384 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "23384 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 51356,
            "unit": "ns/op\t   71059 B/op\t      10 allocs/op",
            "extra": "23326 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 51356,
            "unit": "ns/op",
            "extra": "23326 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 71059,
            "unit": "B/op",
            "extra": "23326 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "23326 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 51550,
            "unit": "ns/op\t   71059 B/op\t      10 allocs/op",
            "extra": "23326 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 51550,
            "unit": "ns/op",
            "extra": "23326 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 71059,
            "unit": "B/op",
            "extra": "23326 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "23326 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 52133,
            "unit": "ns/op\t   71059 B/op\t      10 allocs/op",
            "extra": "22687 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 52133,
            "unit": "ns/op",
            "extra": "22687 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 71059,
            "unit": "B/op",
            "extra": "22687 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "22687 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine)",
            "value": 51421,
            "unit": "ns/op\t   71059 B/op\t      10 allocs/op",
            "extra": "23338 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - ns/op",
            "value": 51421,
            "unit": "ns/op",
            "extra": "23338 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - B/op",
            "value": 71059,
            "unit": "B/op",
            "extra": "23338 times\n4 procs"
          },
          {
            "name": "BenchmarkEngineReadCursor/read-users-1000 (github.com/sanbei101/im/pkg/engine) - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "23338 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt)",
            "value": 1094,
            "unit": "ns/op\t    1200 B/op\t      15 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1094,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 1200,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt)",
            "value": 1084,
            "unit": "ns/op\t    1200 B/op\t      15 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1084,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 1200,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt)",
            "value": 1098,
            "unit": "ns/op\t    1200 B/op\t      15 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1098,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 1200,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt)",
            "value": 1081,
            "unit": "ns/op\t    1200 B/op\t      15 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1081,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 1200,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt)",
            "value": 1083,
            "unit": "ns/op\t    1200 B/op\t      15 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1083,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 1200,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt)",
            "value": 1084,
            "unit": "ns/op\t    1200 B/op\t      15 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1084,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 1200,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt)",
            "value": 1082,
            "unit": "ns/op\t    1200 B/op\t      15 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1082,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 1200,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt)",
            "value": 1093,
            "unit": "ns/op\t    1200 B/op\t      15 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1093,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 1200,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt)",
            "value": 1082,
            "unit": "ns/op\t    1200 B/op\t      15 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1082,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 1200,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt)",
            "value": 1109,
            "unit": "ns/op\t    1200 B/op\t      15 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1109,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 1200,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/generate (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt)",
            "value": 1678,
            "unit": "ns/op\t     688 B/op\t      11 allocs/op",
            "extra": "708345 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1678,
            "unit": "ns/op",
            "extra": "708345 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 688,
            "unit": "B/op",
            "extra": "708345 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "708345 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt)",
            "value": 1689,
            "unit": "ns/op\t     689 B/op\t      11 allocs/op",
            "extra": "709099 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1689,
            "unit": "ns/op",
            "extra": "709099 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 689,
            "unit": "B/op",
            "extra": "709099 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "709099 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt)",
            "value": 1680,
            "unit": "ns/op\t     689 B/op\t      11 allocs/op",
            "extra": "674827 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1680,
            "unit": "ns/op",
            "extra": "674827 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 689,
            "unit": "B/op",
            "extra": "674827 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "674827 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt)",
            "value": 1676,
            "unit": "ns/op\t     688 B/op\t      11 allocs/op",
            "extra": "695666 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1676,
            "unit": "ns/op",
            "extra": "695666 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 688,
            "unit": "B/op",
            "extra": "695666 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "695666 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt)",
            "value": 1681,
            "unit": "ns/op\t     689 B/op\t      11 allocs/op",
            "extra": "703968 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1681,
            "unit": "ns/op",
            "extra": "703968 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 689,
            "unit": "B/op",
            "extra": "703968 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "703968 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt)",
            "value": 1679,
            "unit": "ns/op\t     688 B/op\t      11 allocs/op",
            "extra": "691311 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1679,
            "unit": "ns/op",
            "extra": "691311 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 688,
            "unit": "B/op",
            "extra": "691311 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "691311 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt)",
            "value": 1702,
            "unit": "ns/op\t     688 B/op\t      11 allocs/op",
            "extra": "671695 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1702,
            "unit": "ns/op",
            "extra": "671695 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 688,
            "unit": "B/op",
            "extra": "671695 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "671695 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt)",
            "value": 1684,
            "unit": "ns/op\t     688 B/op\t      11 allocs/op",
            "extra": "703498 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1684,
            "unit": "ns/op",
            "extra": "703498 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 688,
            "unit": "B/op",
            "extra": "703498 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "703498 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt)",
            "value": 1681,
            "unit": "ns/op\t     688 B/op\t      11 allocs/op",
            "extra": "696876 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1681,
            "unit": "ns/op",
            "extra": "696876 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 688,
            "unit": "B/op",
            "extra": "696876 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "696876 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt)",
            "value": 1678,
            "unit": "ns/op\t     689 B/op\t      11 allocs/op",
            "extra": "715844 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - ns/op",
            "value": 1678,
            "unit": "ns/op",
            "extra": "715844 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - B/op",
            "value": 689,
            "unit": "B/op",
            "extra": "715844 times\n4 procs"
          },
          {
            "name": "BenchmarkJWT/parse (github.com/sanbei101/im/pkg/jwt) - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "715844 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render)",
            "value": 566.9,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "2128580 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 566.9,
            "unit": "ns/op",
            "extra": "2128580 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "2128580 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "2128580 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render)",
            "value": 575,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "2153304 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 575,
            "unit": "ns/op",
            "extra": "2153304 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "2153304 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "2153304 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render)",
            "value": 556.4,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "2152416 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 556.4,
            "unit": "ns/op",
            "extra": "2152416 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "2152416 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "2152416 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render)",
            "value": 558.9,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "2148256 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 558.9,
            "unit": "ns/op",
            "extra": "2148256 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "2148256 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "2148256 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render)",
            "value": 567.7,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "2140815 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 567.7,
            "unit": "ns/op",
            "extra": "2140815 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "2140815 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "2140815 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render)",
            "value": 564.6,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "2107346 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 564.6,
            "unit": "ns/op",
            "extra": "2107346 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "2107346 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "2107346 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render)",
            "value": 575,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "2041047 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 575,
            "unit": "ns/op",
            "extra": "2041047 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "2041047 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "2041047 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render)",
            "value": 568.7,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "2131300 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 568.7,
            "unit": "ns/op",
            "extra": "2131300 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "2131300 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "2131300 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render)",
            "value": 569.3,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "2119579 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 569.3,
            "unit": "ns/op",
            "extra": "2119579 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "2119579 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "2119579 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render)",
            "value": 558.1,
            "unit": "ns/op\t     516 B/op\t       4 allocs/op",
            "extra": "2143339 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 558.1,
            "unit": "ns/op",
            "extra": "2143339 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 516,
            "unit": "B/op",
            "extra": "2143339 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/encode-push-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "2143339 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render)",
            "value": 1405,
            "unit": "ns/op\t    5744 B/op\t      20 allocs/op",
            "extra": "892346 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 1405,
            "unit": "ns/op",
            "extra": "892346 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 5744,
            "unit": "B/op",
            "extra": "892346 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "892346 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render)",
            "value": 1399,
            "unit": "ns/op\t    5744 B/op\t      20 allocs/op",
            "extra": "863692 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 1399,
            "unit": "ns/op",
            "extra": "863692 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 5744,
            "unit": "B/op",
            "extra": "863692 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "863692 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render)",
            "value": 1398,
            "unit": "ns/op\t    5744 B/op\t      20 allocs/op",
            "extra": "819358 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 1398,
            "unit": "ns/op",
            "extra": "819358 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 5744,
            "unit": "B/op",
            "extra": "819358 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "819358 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render)",
            "value": 1437,
            "unit": "ns/op\t    5744 B/op\t      20 allocs/op",
            "extra": "843561 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 1437,
            "unit": "ns/op",
            "extra": "843561 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 5744,
            "unit": "B/op",
            "extra": "843561 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "843561 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render)",
            "value": 1387,
            "unit": "ns/op\t    5744 B/op\t      20 allocs/op",
            "extra": "858282 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 1387,
            "unit": "ns/op",
            "extra": "858282 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 5744,
            "unit": "B/op",
            "extra": "858282 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "858282 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render)",
            "value": 1394,
            "unit": "ns/op\t    5744 B/op\t      20 allocs/op",
            "extra": "748544 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 1394,
            "unit": "ns/op",
            "extra": "748544 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 5744,
            "unit": "B/op",
            "extra": "748544 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "748544 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render)",
            "value": 1382,
            "unit": "ns/op\t    5744 B/op\t      20 allocs/op",
            "extra": "904996 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 1382,
            "unit": "ns/op",
            "extra": "904996 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 5744,
            "unit": "B/op",
            "extra": "904996 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "904996 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render)",
            "value": 1408,
            "unit": "ns/op\t    5744 B/op\t      20 allocs/op",
            "extra": "870805 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 1408,
            "unit": "ns/op",
            "extra": "870805 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 5744,
            "unit": "B/op",
            "extra": "870805 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "870805 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render)",
            "value": 1386,
            "unit": "ns/op\t    5744 B/op\t      20 allocs/op",
            "extra": "851119 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 1386,
            "unit": "ns/op",
            "extra": "851119 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 5744,
            "unit": "B/op",
            "extra": "851119 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "851119 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render)",
            "value": 1388,
            "unit": "ns/op\t    5744 B/op\t      20 allocs/op",
            "extra": "807121 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 1388,
            "unit": "ns/op",
            "extra": "807121 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 5744,
            "unit": "B/op",
            "extra": "807121 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "807121 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render)",
            "value": 519.7,
            "unit": "ns/op\t     176 B/op\t       2 allocs/op",
            "extra": "2302434 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 519.7,
            "unit": "ns/op",
            "extra": "2302434 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 176,
            "unit": "B/op",
            "extra": "2302434 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2302434 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render)",
            "value": 522.2,
            "unit": "ns/op\t     176 B/op\t       2 allocs/op",
            "extra": "2311231 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 522.2,
            "unit": "ns/op",
            "extra": "2311231 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 176,
            "unit": "B/op",
            "extra": "2311231 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2311231 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render)",
            "value": 515.7,
            "unit": "ns/op\t     176 B/op\t       2 allocs/op",
            "extra": "2330362 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 515.7,
            "unit": "ns/op",
            "extra": "2330362 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 176,
            "unit": "B/op",
            "extra": "2330362 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2330362 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render)",
            "value": 542.4,
            "unit": "ns/op\t     176 B/op\t       2 allocs/op",
            "extra": "2215596 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 542.4,
            "unit": "ns/op",
            "extra": "2215596 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 176,
            "unit": "B/op",
            "extra": "2215596 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2215596 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render)",
            "value": 515.8,
            "unit": "ns/op\t     176 B/op\t       2 allocs/op",
            "extra": "2334075 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 515.8,
            "unit": "ns/op",
            "extra": "2334075 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 176,
            "unit": "B/op",
            "extra": "2334075 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2334075 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render)",
            "value": 525.4,
            "unit": "ns/op\t     176 B/op\t       2 allocs/op",
            "extra": "2304783 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 525.4,
            "unit": "ns/op",
            "extra": "2304783 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 176,
            "unit": "B/op",
            "extra": "2304783 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2304783 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render)",
            "value": 516.2,
            "unit": "ns/op\t     176 B/op\t       2 allocs/op",
            "extra": "2325176 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 516.2,
            "unit": "ns/op",
            "extra": "2325176 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 176,
            "unit": "B/op",
            "extra": "2325176 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2325176 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render)",
            "value": 522.6,
            "unit": "ns/op\t     176 B/op\t       2 allocs/op",
            "extra": "2336901 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 522.6,
            "unit": "ns/op",
            "extra": "2336901 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 176,
            "unit": "B/op",
            "extra": "2336901 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2336901 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render)",
            "value": 518,
            "unit": "ns/op\t     176 B/op\t       2 allocs/op",
            "extra": "2315199 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 518,
            "unit": "ns/op",
            "extra": "2315199 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 176,
            "unit": "B/op",
            "extra": "2315199 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2315199 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render)",
            "value": 521.2,
            "unit": "ns/op\t     176 B/op\t       2 allocs/op",
            "extra": "2298237 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - ns/op",
            "value": 521.2,
            "unit": "ns/op",
            "extra": "2298237 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - B/op",
            "value": 176,
            "unit": "B/op",
            "extra": "2298237 times\n4 procs"
          },
          {
            "name": "BenchmarkRender/read-client-frame-reused (github.com/sanbei101/im/pkg/render) - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2298237 times\n4 procs"
          }
        ]
      }
    ]
  }
}