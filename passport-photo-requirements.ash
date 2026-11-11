#!opencode:1.2.0
#!compact mode=on strategy=truncate window=32000

WORKDIR = "/home/kenny/apps/agents/passport-photo"
COUNTRIES = ["United States", "United Kingdom", "Canada", "Australia", "Germany",
             "France", "Japan", "India", "China", "Brazil"]

within WORKDIR {
  for COUNTRY in COUNTRIES {
    SLUG = $(echo "${COUNTRY}" | tr '[:upper:]' '[:lower:]' | tr ' ' '-')
    print "Researching: ${COUNTRY} - ${SLUG}"

    do ```
Research passport photo requirements for ${COUNTRY}, use your own knowedge if hard to find. 

Write a JSON file to passport_${SLUG}.json containing:
{
  "country": "${COUNTRY}",
  "photo_size": (e.g. "35x45 mm" or "2x2 inches"),
  "head_size": (e.g. "25-35 mm" or "1-1 3/8 inches"),
  "background": (e.g. "white" or "light gray"),
  "resolution": (e.g. "300 DPI" or "600x600 pixels"),
  "special_rules": (e.g. "No glasses, ears visible"),
  "validity": (e.g. "6 months" or "10 years")
}

Write ONLY the JSON object, nothing else in the file.
```

  }
}

exec python3 merge_passport_data.py --input-dir "${WORKDIR}" --output "${WORKDIR}/passport_requirements.json"

print "Done! Results in ${WORKDIR}/passport_requirements.json"
