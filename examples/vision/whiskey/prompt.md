**Task:**

Analyze the provided images of liquor bottles and extract all relevant textual
and visual information from the bottles' labels, engravings, or other markings.
The extracted data should be structured according to the given JSON schema.

---

### **Instructions for Data Extraction:**

1. **Handling Multiple Images & Bottles:**

   - The provided images may include **multiple pictures of the same bottle from
     different angles** or **images of different bottles**.
   - Group images that represent the same bottle together, ensuring all
     extracted details are consolidated into a **single entry per bottle**.
   - If images contain **different bottles**, treat them as separate entries and
     return multiple structured results.
2. **Textual Information Extraction:**

   - Extract all **text appearing on the bottle**, including brand name, liquor
     type, alcohol content, volume, distillery details, and any additional
     relevant details.
   - If the text is in a **non-English language**, detect the language and
     provide both the **original text** and its **English translation** where
     applicable.
   - Identify and record any **batch numbers, bottle numbers, serial numbers,
     barcodes, or certification marks**.
3. **Liquor Classification & Attributes:**

   - Identify and categorize the **type of liquor** (e.g., Whiskey, Bourbon,
     Tequila, Vodka, etc.).
   - Detect subcategories such as **Reposado, Anejo, Single Malt, Cask Strength,
     etc.**.
   - Extract **alcohol by volume (ABV%)** and **bottle volume (e.g., 750ml,
     1L)**.
   - Capture details on **aging process, ingredients, and distillation methods**
     if present.
4. **Multilingual Label Handling:**

   - If the label contains **multiple languages**, extract all relevant text and
     specify the detected languages.
   - Provide translations where necessary while keeping the original text for
     reference.
5. **Certification & Legal Compliance:**

   - Identify any official **certification marks or legal designations** (e.g.,
     DOC, Organic, Kosher, Government Warning Labels).
   - Capture **geographical indications** or protected origin status (e.g.,
     “Scotch Whisky,” “Mezcal de Oaxaca”).
6. **Bottle Story & Additional Context:**

   - If the bottle includes **historical or descriptive text** about its
     production, origin, or significance, extract it as a "bottle story".
   - Capture **marketing descriptions, unique production processes, or special
     limited-edition details**.
7. **Barcode & Additional Identifiers:**

   - Extract any **barcodes, QR codes, or serial numbers** that might be present
     on the bottle.
   - If applicable, provide an OCR-readable version of these identifiers.
8. **Image-Based Attributes (If Detectable):**

   - Identify **distinctive bottle shapes, label designs, embossed text, or
     branding elements**.
   - Note any **visual quality markers**, such as wax seals, numbered editions,
     or hand-signed elements.
9. **Output Structure & Consistency:**

   - **Return multiple results** in JSON format, where each detected bottle is a
     separate entry.
   - **Consolidate details for the same bottle** when multiple images represent
     the same item.
   - Format extracted data according to the **provided JSON schema**.
   - Ensure **consistency in naming conventions**, units (e.g., ml for volume, %
     for ABV), and categorical classification.
   - If a field is missing or unreadable, set its value to `null` instead of
     omitting it.

---

### **Key Considerations for Accuracy:**

- **Handle multiple images efficiently**: Determine if images show different
  angles of the same bottle or entirely different bottles.
- **Ensure high OCR accuracy**: Prioritize clear text recognition for label
  details.
- **Avoid misclassification**: Validate liquor type and subtypes based on known
  industry terminology.
- **Retain structured formatting**: Follow the JSON schema strictly for seamless
  API integration.
