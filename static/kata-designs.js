// Authored kata designs, one per creature: body type, vibe (cute or cool),
// color scheme as hues, pattern, and the signature features that make the
// name mean something. Regenerate nothing: edit by hand.
// Consumed by static/kata-svg.js via creature(seed, { design, grade }).
const KATA_DESIGNS = {
 "g1-short-vowels": {
  "body": "sprout",
  "vibe": "cute",
  "hue": 95,
  "accent": 340,
  "pattern": "none",
  "features": [
   "tuft"
  ],
  "eyes": "big",
  "grade": 1
 },
 "g1-digraphs": {
  "body": "ghost",
  "vibe": "cute",
  "hue": 262,
  "accent": 45,
  "pattern": "none",
  "features": [
   "halo"
  ],
  "mouth": "shh",
  "eyes": "sleepy",
  "grade": 1
 },
 "g1-blends": {
  "body": "beast",
  "vibe": "cool",
  "hue": 26,
  "accent": 200,
  "pattern": "stripes",
  "features": [
   "pointEars",
   "tailTip"
  ],
  "grade": 1
 },
 "g1-final-e": {
  "body": "wisp",
  "vibe": "cute",
  "hue": 212,
  "accent": 48,
  "mood": "light",
  "pattern": "none",
  "features": [
   "crystals"
  ],
  "eyes": "sleepy",
  "mouth": "flat",
  "grade": 1
 },
 "g1-vowel-teams": {
  "body": "blob",
  "vibe": "cute",
  "hue": 332,
  "accent": 170,
  "pattern": "patches",
  "features": [
   "twinHead",
   "roundEars"
  ],
  "eyes": "big",
  "grade": 1
 },
 "g1-irregular": {
  "body": "turtle",
  "vibe": "cute",
  "hue": 352,
  "accent": 150,
  "pattern": "none",
  "features": [
   "shellPlates"
  ],
  "mouth": "open",
  "grade": 1
 },
 "g2-vowel-teams": {
  "body": "bird",
  "vibe": "cute",
  "hue": 38,
  "accent": 112,
  "pattern": "none",
  "features": [
   "leafCrest"
  ],
  "grade": 2
 },
 "g2-r-controlled": {
  "body": "beast",
  "vibe": "cool",
  "hue": 14,
  "accent": 52,
  "pattern": "none",
  "features": [
   "mane",
   "curvedHorns"
  ],
  "mouth": "grin",
  "grade": 2
 },
 "g2-two-syllable": {
  "body": "blob",
  "vibe": "cute",
  "hue": 188,
  "accent": 28,
  "pattern": "split",
  "features": [
   "antennae",
   "roundEars"
  ],
  "grade": 2
 },
 "g2-endings": {
  "body": "fish",
  "vibe": "cute",
  "hue": 204,
  "accent": 40,
  "pattern": "spots",
  "features": [
   "bigTail",
   "dorsalSail"
  ],
  "grade": 2
 },
 "g2-irregular": {
  "body": "bug",
  "vibe": "cute",
  "hue": 58,
  "accent": 290,
  "pattern": "checks",
  "features": [
   "antennae",
   "star"
  ],
  "eyes": "big",
  "mouth": "grin",
  "grade": 2
 },
 "g3-multisyllable": {
  "body": "serpent",
  "vibe": "cool",
  "hue": 142,
  "accent": 320,
  "pattern": "stripes",
  "features": [
   "tailSpade"
  ],
  "grade": 3
 },
 "g3-prefixes": {
  "body": "beast",
  "vibe": "cool",
  "hue": 28,
  "accent": 200,
  "pattern": "none",
  "features": [
   "pointEars",
   "bigTail",
   "scarf"
  ],
  "mouth": "cat",
  "grade": 3
 },
 "g3-suffixes": {
  "body": "fish",
  "vibe": "cool",
  "hue": 248,
  "accent": 60,
  "mood": "dark",
  "pattern": "stripes",
  "features": [
   "fins",
   "bubbles"
  ],
  "grade": 3
 },
 "g3-spelling-changes": {
  "body": "octo",
  "vibe": "cute",
  "hue": 282,
  "accent": 120,
  "pattern": "patches",
  "features": [
   "antennae"
  ],
  "grade": 3
 },
 "g3-irregular": {
  "body": "beast",
  "vibe": "cool",
  "hue": 302,
  "accent": 52,
  "mood": "dark",
  "pattern": "none",
  "features": [
   "mask",
   "pointEars",
   "tailTip"
  ],
  "mouth": "grin",
  "grade": 3
 },
 "g4-multisyllable": {
  "body": "golem",
  "vibe": "cute",
  "hue": 214,
  "accent": 30,
  "pattern": "checks",
  "features": [
   "mohawk"
  ],
  "grade": 4
 },
 "g4-tricky": {
  "body": "dragon",
  "vibe": "cool",
  "hue": 98,
  "accent": 340,
  "mood": "dark",
  "pattern": "none",
  "features": [
   "tusks",
   "backSpikes"
  ],
  "mouth": "fang",
  "grade": 4
 },
 "g4-roots": {
  "body": "sprout",
  "vibe": "cool",
  "hue": 24,
  "accent": 130,
  "mood": "dark",
  "pattern": "none",
  "features": [
   "antlers"
  ],
  "eyes": "sleepy",
  "mouth": "flat",
  "grade": 4
 },
 "g4-suffixes": {
  "body": "dragon",
  "vibe": "cool",
  "hue": 350,
  "accent": 50,
  "pattern": "none",
  "features": [
   "tailSpade",
   "curvedHorns"
  ],
  "grade": 4
 },
 "g4-academic": {
  "body": "bird",
  "vibe": "cool",
  "hue": 232,
  "accent": 48,
  "pattern": "none",
  "features": [
   "glasses",
   "crestFeathers"
  ],
  "eyes": "round",
  "mouth": "flat",
  "grade": 4
 },
 "g5-greek-latin": {
  "body": "wisp",
  "vibe": "cool",
  "hue": 196,
  "accent": 300,
  "mood": "light",
  "pattern": "none",
  "features": [
   "halo",
   "crystals"
  ],
  "eyes": "sharp",
  "grade": 5
 },
 "g5-suffixes": {
  "body": "beast",
  "vibe": "cool",
  "hue": 44,
  "accent": 10,
  "pattern": "none",
  "features": [
   "mane",
   "roundEars"
  ],
  "mouth": "grin",
  "grade": 5
 },
 "g5-multisyllable": {
  "body": "dragon",
  "vibe": "cool",
  "hue": 122,
  "accent": 40,
  "pattern": "stripes",
  "features": [
   "backSpikes",
   "horns"
  ],
  "grade": 5
 },
 "g5-academic": {
  "body": "bird",
  "vibe": "cool",
  "hue": 270,
  "accent": 45,
  "pattern": "none",
  "features": [
   "glasses",
   "plume"
  ],
  "eyes": "round",
  "grade": 5
 },
 "g5-irregular": {
  "body": "octo",
  "vibe": "cool",
  "hue": 286,
  "accent": 160,
  "mood": "dark",
  "pattern": "none",
  "features": [
   "thirdEye",
   "crystals"
  ],
  "eyes": "sharp",
  "mouth": "flat",
  "grade": 5
 },
 "sight-g1": {
  "body": "bug",
  "vibe": "cute",
  "hue": 54,
  "accent": 200,
  "pattern": "none",
  "features": [
   "wingsFeather",
   "lantern"
  ],
  "eyes": "big",
  "grade": 1
 },
 "sight-g2": {
  "body": "wisp",
  "vibe": "cute",
  "hue": 22,
  "accent": 48,
  "pattern": "none",
  "features": [
   "star"
  ],
  "eyes": "big",
  "grade": 2
 },
 "sight-g3": {
  "body": "turtle",
  "vibe": "cool",
  "hue": 202,
  "accent": 48,
  "pattern": "checks",
  "features": [
   "lantern"
  ],
  "grade": 3
 },
 "sight-g4": {
  "body": "fish",
  "vibe": "cool",
  "hue": 228,
  "accent": 55,
  "mood": "dark",
  "pattern": "none",
  "features": [
   "lantern",
   "bubbles"
  ],
  "eyes": "big",
  "grade": 4
 },
 "sight-g5": {
  "body": "ghost",
  "vibe": "cool",
  "hue": 170,
  "accent": 300,
  "pattern": "none",
  "features": [
   "halo",
   "wingsFeather"
  ],
  "eyes": "sharp",
  "mouth": "smile",
  "grade": 5
 },
 "math-addsub-g1": {
  "body": "beast",
  "vibe": "cute",
  "hue": 36,
  "accent": 200,
  "pattern": "stripes",
  "features": [
   "floppyEars",
   "stubbyTail"
  ],
  "mouth": "open",
  "grade": 1
 },
 "math-addsub-g2": {
  "body": "beast",
  "vibe": "cool",
  "hue": 20,
  "accent": 190,
  "pattern": "stripes",
  "features": [
   "pointEars",
   "bigTail"
  ],
  "grade": 2
 },
 "math-addsub-g3": {
  "body": "beast",
  "vibe": "cool",
  "hue": 216,
  "accent": 40,
  "mood": "dark",
  "pattern": "stripes",
  "features": [
   "pointEars",
   "tuft",
   "tailTip"
  ],
  "eyes": "sharp",
  "grade": 3
 },
 "math-addsub-g4": {
  "body": "blob",
  "vibe": "cute",
  "hue": 26,
  "accent": 200,
  "mood": "dark",
  "pattern": "stripes",
  "features": [
   "roundEars",
   "scarf"
  ],
  "grade": 4
 },
 "math-addsub-g5": {
  "body": "dragon",
  "vibe": "cool",
  "hue": 4,
  "accent": 52,
  "pattern": "stripes",
  "features": [
   "horns",
   "backSpikes"
  ],
  "grade": 5
 },
 "math-mul-g1": {
  "body": "blob",
  "vibe": "cute",
  "hue": 166,
  "accent": 330,
  "pattern": "checks",
  "features": [
   "antennae"
  ],
  "eyes": "big",
  "grade": 1
 },
 "math-mul-g2": {
  "body": "blob",
  "vibe": "cute",
  "hue": 342,
  "accent": 180,
  "pattern": "checks",
  "features": [
   "longEars"
  ],
  "mouth": "cat",
  "grade": 2
 },
 "math-mul-g3": {
  "body": "beast",
  "vibe": "cute",
  "hue": 46,
  "accent": 210,
  "pattern": "checks",
  "features": [
   "pointEars",
   "tailTip",
   "headband"
  ],
  "mouth": "cat",
  "grade": 3
 },
 "math-mul-g4": {
  "body": "beast",
  "vibe": "cool",
  "hue": 182,
  "accent": 30,
  "pattern": "checks",
  "features": [
   "pointEars",
   "tuft",
   "ruff"
  ],
  "eyes": "sharp",
  "grade": 4
 },
 "math-mul-g5": {
  "body": "beast",
  "vibe": "cool",
  "hue": 30,
  "accent": 210,
  "pattern": "checks",
  "features": [
   "pointEars",
   "bigTail"
  ],
  "mouth": "fang",
  "grade": 5
 },
 "math-div-g1": {
  "body": "sprout",
  "vibe": "cute",
  "hue": 132,
  "accent": 20,
  "pattern": "split",
  "features": [
   "leafCrest"
  ],
  "grade": 1
 },
 "math-div-g2": {
  "body": "fish",
  "vibe": "cute",
  "hue": 258,
  "accent": 50,
  "pattern": "split",
  "features": [
   "fins",
   "bigTail"
  ],
  "grade": 2
 },
 "math-div-g3": {
  "body": "beast",
  "vibe": "cool",
  "hue": 16,
  "accent": 190,
  "mood": "dark",
  "pattern": "split",
  "features": [
   "pointEars",
   "bigTail",
   "mask"
  ],
  "grade": 3
 },
 "math-div-g4": {
  "body": "bird",
  "vibe": "cool",
  "hue": 240,
  "accent": 48,
  "pattern": "split",
  "features": [
   "tuft",
   "wingsFeather"
  ],
  "eyes": "big",
  "grade": 4
 },
 "math-div-g5": {
  "body": "dragon",
  "vibe": "cool",
  "hue": 266,
  "accent": 44,
  "pattern": "split",
  "features": [
   "curvedHorns",
   "tailSpade"
  ],
  "grade": 5
 },
 "math-frac-g1": {
  "body": "blob",
  "vibe": "cute",
  "hue": 300,
  "accent": 60,
  "pattern": "patches",
  "features": [
   "crown"
  ],
  "eyes": "big",
  "grade": 1
 },
 "math-frac-g2": {
  "body": "bird",
  "vibe": "cute",
  "hue": 50,
  "accent": 300,
  "pattern": "patches",
  "features": [
   "wingsFeather",
   "plume"
  ],
  "grade": 2
 },
 "math-frac-g3": {
  "body": "bird",
  "vibe": "cool",
  "hue": 10,
  "accent": 200,
  "pattern": "patches",
  "features": [
   "crestFeathers",
   "headband"
  ],
  "eyes": "sharp",
  "grade": 3
 },
 "math-frac-g4": {
  "body": "golem",
  "vibe": "cute",
  "hue": 32,
  "accent": 180,
  "mood": "dark",
  "pattern": "patches",
  "features": [
   "roundEars",
   "cape"
  ],
  "grade": 4
 },
 "math-frac-g5": {
  "body": "bug",
  "vibe": "cool",
  "hue": 284,
  "accent": 50,
  "mood": "light",
  "pattern": "patches",
  "features": [
   "wingsFeather",
   "antennae",
   "crystals"
  ],
  "grade": 5
 }
};
if (typeof module !== "undefined" && module.exports) module.exports = KATA_DESIGNS;
