{
  "targets": [
    {
      "target_name": "compare",
      "include_dirs" : [
        "<!(node -e \"require('nan')\")"
      ],
      "sources": [ "src/compare.cpp"]
    },
    {
      "target_name": "findapng",
      "include_dirs" : [
        "<!(node -e \"require('nan')\")"
      ],
      "sources": [ "src/findapng.cpp","src/apngDetector.cpp"]
    }
  ]
}
