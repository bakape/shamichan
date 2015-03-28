{
    "targets": [
        {
            "target_name": "compare",
            "include_dirs" : [
                "<!(node -e \"require('nan')\")"
            ],
            "sources": [ "compare.cpp"]
        },
        {
            "target_name": "findapng",
            "include_dirs" : [
                "<!(node -e \"require('nan')\")"
            ],
            "sources": [ "findapng.cpp","apngDetector.cpp"]
        }
    ]
}
