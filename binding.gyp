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
    },
    {
      "target_name": "mnemonics",
      "include_dirs" : [
        "<!(node -e \"require('nan')\")",
        "<(node_root_dir)/deps/openssl/openssl/include"
      ],
      "sources": ["src/mnemonics.cpp","src/mnemonizer.cpp"],
      "conditions": [
        ["target_arch=='ia32'", {
          "include_dirs": [ "<(node_root_dir)/deps/openssl/config/piii" ]
        }],
        ["target_arch=='x64'", {
          "include_dirs": [ "<(node_root_dir)/deps/openssl/config/k8" ]
        }],
        ["target_arch=='arm'", {
          "include_dirs": [ "<(node_root_dir)/deps/openssl/config/arm" ]
        }]
      ]
    },
    {
      "target_name": "tripcode",
      "include_dirs" : [
       "<!(node -e \"require('nan')\")"
      ],
      "sources": ["src/tripcode.cc"],
      "link_settings": {
        "conditions": [
          [
            "OS==\"linux\"",
            {"libraries": ["-lcrypt"]}
          ],
          [
            "OS==\"freebsd\"",
            {"libraries": ["-lcrypt"]}
          ],
          [
            "OS==\"mac\"",
            {"libraries": ["-lcrypto", "/usr/lib/libiconv.dylib"]}
          ]
        ]
      }
    }
  ]
}
