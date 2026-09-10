# SkillDojo Spelling Curriculum

Status: initial product curriculum for educator review. It is a structured
starting point, not a replacement for a school or intervention curriculum.

## Instructional principles

1. Organize practice around transferable spelling concepts, not word length.
2. Connect phonology (sounds), orthography (letters), morphology (meaningful
   word parts), meaning, and usage.
3. Include both regular pattern words and high-frequency irregular words.
4. Ask learners to retrieve spellings and provide immediate corrective
   feedback.
5. Review previously difficult words over time rather than treating one
   successful round as mastery.
6. Use supportive language. Errors identify what to teach next.

This approach is informed by:

- [Common Core Language standards](https://www.thecorestandards.org/ELA-Literacy/L/)
  for grade-level spelling, patterns, suffixes, and word parts.
- [Common Core Foundational Skills](https://www.thecorestandards.org/ELA-Literacy/RF/)
  for phonics, syllabication, morphology, and multisyllable word analysis.
- The [UFLI Foundations scope and sequence](https://ufli.education.ufl.edu/foundations/)
  for a systematic progression through phonics and spelling concepts.
- The Institute of Education Sciences guide,
  [Foundational Skills to Support Reading for Understanding](https://ies.ed.gov/ncee/wwc/PracticeGuide/21/Published),
  for linking speech sounds to letters and teaching learners to analyze,
  recognize, and write words.
- The [Children's Picture Books Lexicon](https://pmc.ncbi.nlm.nih.gov/articles/PMC11289352/)
  and [Children's Printed Word Database](https://pubmed.ncbi.nlm.nih.gov/20021708/)
  as candidate sources for child-centered frequency and lexical attributes.

## Initial Grade 1–5 scope

The scope describes the primary concept being practiced. Skills should be
reviewed cumulatively, and placement should ultimately respond to demonstrated
knowledge rather than age alone.

| Grade | Initial focus areas | Intended transfer |
| --- | --- | --- |
| 1 | Short vowels in CVC words; consonant digraphs; consonant blends; silent e; common vowel teams; high-frequency heart words | Segment simple words, connect phonemes to graphemes, and distinguish regular from unexpected parts |
| 2 | Broader vowel teams; r-controlled vowels; two-syllable words; inflectional endings; high-frequency irregular words | Generalize learned patterns and preserve recognizable base words |
| 3 | Multisyllable words; prefixes; suffixes; doubling/drop-e/change-y rules; common tricky words | Use syllables and meaningful parts rather than memorizing whole strings |
| 4 | Multisyllable analysis; commonly misspelled words; roots and prefixes; advanced suffixes; academic vocabulary | Combine sound, syllable, and morphology strategies |
| 5 | Greek and Latin parts; complex suffixes; long multisyllable words; academic vocabulary; advanced irregular words | Analyze unfamiliar academic and domain-specific words independently |

## Session composition

### Mixed practice

Mixed sessions rotate across the grade's focus areas. The selector uses
round-robin sampling so short sessions contain genuine variety rather than a
random cluster from one category.

### Focused practice

- Five-word session: five examples of the selected concept.
- Ten-word session: five selected-concept words followed by five mixed review
  words.

### Sight-word practice

Sight-word sessions contain only high-frequency words: five words in a short
session and ten in a long session. The initial 50 entries come from the leading
non-contraction words in the 2024 Children's Picture Book Sight Words ranking,
divided into progressive frequency bands. Above Grade 1, a session draws 60%
from the selected grade's new band and 40% cumulatively from earlier bands.
Source rank is retained on every entry. The single-letter pronoun `I` was
omitted because it does not provide meaningful spelling practice in this game.

"Sight word" means a word recognized automatically; it does not mean every word
must be memorized as an indivisible shape. Regular sound-letter relationships
should still be mapped, while genuinely unexpected parts receive heart-word
instruction. This distinction follows the
[UFLI glossary](https://ufli.education.ufl.edu/resources/teaching-resources/glossary/)
and its guidance for
[irregular and high-frequency words](https://ufli.education.ufl.edu/resources/teaching-resources/instructional-activities/irregular-and-high-frequency-words/).

Future adaptive sessions should replace fixed proportions with learner data:
recent errors, time since last successful retrieval, and concept mastery.

## Current word record

```js
{
  word: "running",
  skill: "g3-spelling-changes",
  clue: "Moving quickly on foot",
  sentence: "They were running around the track."
}
```

Planned fields for an educator-reviewed production bank:

```js
{
  locale: "en-US",
  phonemes: ["r", "uh", "n", "ih", "ng"],
  graphemes: ["r", "u", "nn", "i", "ng"],
  syllables: ["run", "ning"],
  morphemes: ["run", "ing"],
  baseWord: "run",
  regularity: "rule-governed",
  trickyPart: "Double n before adding -ing.",
  frequencySource: "...",
  frequencyBand: "high",
  curriculumSources: ["..."],
  reviewedBy: "...",
  reviewedAt: "YYYY-MM-DD"
}
```

## Word acceptance checklist

Before a word enters the reviewed bank:

- The target concept is identifiable and appropriate for the sequence.
- Earlier concepts required by the word have already been introduced.
- The definition is concrete, accurate, and understandable at the target grade.
- The sentence makes the intended meaning clear without sounding artificial.
- The sentence contains the exact answer once so it can be safely blanked.
- American English spelling and pronunciation are used for the current locale.
- Proper nouns, stigmatizing language, and culturally narrow assumptions are
  avoided unless instruction specifically requires them.
- Ambiguous homophones have enough sentence context to identify the answer.
- Speech synthesis pronounces the word acceptably on supported target devices.
- The entry has passed content checks and educator review.

## Research and expansion workflow

1. Choose the target locale and standards alignment. The current product uses
   US English and Grades 1–5.
2. Compare at least one systematic phonics sequence, applicable state or
   district expectations, and the Common Core skills where relevant.
3. Generate candidate words from child-centered corpora and authentic
   elementary reading materials. Frequency is a filter, not the curriculum.
4. Tag candidates for spelling concept, prerequisite concepts, morphology,
   frequency, regularity, and meaning.
5. Balance each concept with familiar examples, contrast words, review words,
   and novel transfer words.
6. Have an elementary literacy specialist review placement, explanations, and
   language.
7. Pilot with children and observe incorrect spellings, clue comprehension,
   pronunciation, engagement, and transfer to new words.
8. Revise the scope and difficulty from evidence. Do not move a word solely
   because it is long or because many players miss it.

Before importing a published curriculum's selection or arrangement, verify its
license. Facts and individual words may be broadly usable, while a curated
sequence, definitions, examples, and other authored materials can have separate
rights.

## Product roadmap

### Phase 1 — implemented

- Skill taxonomy for Grades 1–5.
- At least five words per instructional focus.
- Mixed and focused session generation.
- A visible concept label and post-round spelling lesson.
- Automated integrity checks for word and skill data.
- Fifty ranked sight-word entries and a dedicated Sight words focus.

### Phase 2 — stronger retrieval

- Hear-and-spell mode requiring the full word before hints.
- A no-hints retype after Word Rescue reveals the answer.
- Grapheme tiles or sound boxes for early grades.
- Word-part construction for prefixes, roots, and suffixes.

### Phase 3 — adaptive review

- Store learning history locally without accounts.
- Revisit missed words using spaced retrieval.
- Track mastery separately by word and spelling concept.
- Add unseen transfer words to determine whether a pattern generalized.

### Phase 4 — content scale and validation

- Expand toward 150–250 educator-reviewed words per grade.
- Add source and review provenance to every entry.
- Pilot pronunciation and wording across common devices.
- Support additional English locales without silently mixing conventions.
