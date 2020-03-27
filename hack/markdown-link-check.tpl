{
    "ignorePatterns": [
        {
            "pattern": "^(http|https|ftp|mailto):"
        },
        {
            "pattern": "\\.\\./media/"
        },
        {
            "comment": "anchors to current file are ignored",
            "pattern": "^#.+$"
        }
    ],
    "replacementPatterns": [
        {
            "comment": "remove anchor part",
            "pattern": "#.+$",
            "replacement": ""
        },
        {
            "comment": "prefix with repo root",
            "pattern": "^/media",
            "replacement": "<ROOT>/media"
        }
    ]
}
