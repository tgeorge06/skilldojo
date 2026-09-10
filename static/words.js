// SkillDojo's spelling curriculum is organized by teachable patterns instead
// of word length alone. Each focus contains at least five words so it can power
// a complete focused session; ten-word sessions add mixed review words.
const SPELLING_SKILLS = {
  1: [
    { id: "g1-short-vowels", label: "Short vowels", tip: "Listen for the short vowel sound in the middle of a simple word." },
    { id: "g1-digraphs", label: "Consonant teams", tip: "Two consonants can work together to make one sound, as in sh, ch, th, and wh." },
    { id: "g1-blends", label: "Consonant blends", tip: "In a blend, you can still hear each consonant sound." },
    { id: "g1-final-e", label: "Silent e", tip: "A silent e can make the earlier vowel say its name." },
    { id: "g1-vowel-teams", label: "Vowel teams", tip: "Two vowels can work together to represent one vowel sound." },
    { id: "g1-irregular", label: "Heart words", tip: "Some common words have an unexpected part that must be learned by heart." },
  ],
  2: [
    { id: "g2-vowel-teams", label: "Advanced vowel teams", tip: "Vowel teams can represent long vowels and other sounds; notice the whole team." },
    { id: "g2-r-controlled", label: "R-controlled vowels", tip: "A vowel followed by r makes a special sound, as in ar, er, ir, or, and ur." },
    { id: "g2-two-syllable", label: "Two-syllable words", tip: "Say each syllable slowly, then spell and join the parts." },
    { id: "g2-endings", label: "Word endings", tip: "Look for a base word before adding endings such as -ing, -ed, -es, or -s." },
    { id: "g2-irregular", label: "High-frequency heart words", tip: "Map the regular sounds first, then remember the unexpected letters by heart." },
  ],
  3: [
    { id: "g3-multisyllable", label: "Longer words", tip: "Break a longer word into spoken syllables before spelling each part." },
    { id: "g3-prefixes", label: "Prefixes", tip: "A prefix is a meaningful part added to the beginning of a base word." },
    { id: "g3-suffixes", label: "Suffixes", tip: "A suffix is a meaningful ending; find the base word before adding it." },
    { id: "g3-spelling-changes", label: "Suffix spelling changes", tip: "Adding a suffix can double a consonant, drop silent e, or change y to i." },
    { id: "g3-irregular", label: "Common tricky words", tip: "Pronounce the word, identify its regular parts, and study only the unexpected part." },
  ],
  4: [
    { id: "g4-multisyllable", label: "Multisyllable words", tip: "Use syllable boundaries to spell one meaningful chunk at a time." },
    { id: "g4-tricky", label: "Commonly misspelled words", tip: "Find the surprising letter sequence and connect it to a memory clue." },
    { id: "g4-roots", label: "Roots and prefixes", tip: "Recognizing roots and prefixes makes long words easier to analyze and spell." },
    { id: "g4-suffixes", label: "Advanced suffixes", tip: "Keep the base word in mind while checking whether its spelling changes." },
    { id: "g4-academic", label: "Academic vocabulary", tip: "Connect spelling, meaning, and use so the word becomes useful in real writing." },
  ],
  5: [
    { id: "g5-greek-latin", label: "Greek and Latin parts", tip: "Greek and Latin roots carry stable meanings and often stable spellings." },
    { id: "g5-suffixes", label: "Complex suffixes", tip: "Separate the base or root from endings such as -tion, -ous, -able, and -ity." },
    { id: "g5-multisyllable", label: "Complex multisyllable words", tip: "Combine syllable, sound, and meaningful-part strategies for long words." },
    { id: "g5-academic", label: "Academic vocabulary", tip: "Study the word's pronunciation, structure, meaning, and use together." },
    { id: "g5-irregular", label: "Advanced tricky words", tip: "Mark the unexpected letter sequence and retrieve the full spelling from memory." },
  ],
};

const spellingWord = (word, skill, clue, sentence) => ({ word, skill, clue, sentence });

const SPELLING_WORDS = {
  1: [
    spellingWord("cat", "g1-short-vowels", "A small pet that purrs", "The cat naps in a sunny spot."),
    spellingWord("dog", "g1-short-vowels", "A pet that barks", "The dog runs after the red ball."),
    spellingWord("sun", "g1-short-vowels", "The bright star we see in the sky", "The sun warms the playground."),
    spellingWord("pet", "g1-short-vowels", "An animal cared for at home", "Our pet likes to play with us."),
    spellingWord("pig", "g1-short-vowels", "A farm animal with a snout", "The pink pig rolled in the mud."),

    spellingWord("fish", "g1-digraphs", "An animal that swims and has fins", "The fish glides through the clear water."),
    spellingWord("ship", "g1-digraphs", "A large boat", "The ship sailed across the bay."),
    spellingWord("chat", "g1-digraphs", "To talk in a friendly way", "We like to chat during lunch."),
    spellingWord("thin", "g1-digraphs", "Not thick", "The thin paper folded easily."),
    spellingWord("when", "g1-digraphs", "A question word about time", "When will the movie begin?"),

    spellingWord("frog", "g1-blends", "A small animal that hops and croaks", "The green frog sits beside the pond."),
    spellingWord("jump", "g1-blends", "To push off the ground with your feet", "Can you jump over the little puddle?"),
    spellingWord("stop", "g1-blends", "To end movement or an action", "The bus will stop at the corner."),
    spellingWord("hand", "g1-blends", "The part at the end of your arm", "Raise your hand if you know the answer."),
    spellingWord("flag", "g1-blends", "A piece of cloth used as a symbol", "The flag waved in the wind."),

    spellingWord("cake", "g1-final-e", "A sweet baked food", "We shared a cake at the party."),
    spellingWord("home", "g1-final-e", "The place where you live", "We walked home after school."),
    spellingWord("bike", "g1-final-e", "A vehicle with two wheels", "My bike has a silver bell."),
    spellingWord("cute", "g1-final-e", "Pretty or lovable", "The puppy looked cute in its tiny bed."),
    spellingWord("game", "g1-final-e", "An activity played for fun", "We learned a new card game."),

    spellingWord("tree", "g1-vowel-teams", "A tall plant with a trunk and leaves", "A bird made its nest in the tree."),
    spellingWord("rain", "g1-vowel-teams", "Water drops that fall from clouds", "The rain taps softly on the window."),
    spellingWord("book", "g1-vowel-teams", "Pages with a story inside", "We read a funny book before bed."),
    spellingWord("seed", "g1-vowel-teams", "A small part of a plant that can grow", "We planted the seed in soft soil."),
    spellingWord("boat", "g1-vowel-teams", "A small vessel that travels on water", "The blue boat floated near the dock."),

    spellingWord("said", "g1-irregular", "The past tense of say", "Mia said hello to her new neighbor."),
    spellingWord("have", "g1-irregular", "To own or hold something", "I have two crayons in my pocket."),
    spellingWord("come", "g1-irregular", "To move toward a place", "Please come sit beside me."),
    spellingWord("some", "g1-irregular", "An amount that is not all", "Would you like some apple slices?"),
    spellingWord("give", "g1-irregular", "To hand something to someone", "I will give you the first turn."),
  ],
  2: [
    spellingWord("yellow", "g2-vowel-teams", "The color of lemons and sunshine", "She wore a bright yellow hat."),
    spellingWord("school", "g2-vowel-teams", "A place where students learn", "Our school has a big library."),
    spellingWord("clean", "g2-vowel-teams", "Free from dirt", "We helped clean the paintbrushes."),
    spellingWord("night", "g2-vowel-teams", "The dark part of each day", "The moon shone at night."),
    spellingWord("play", "g2-vowel-teams", "To have fun with a game or activity", "We play outside after lunch."),

    spellingWord("winter", "g2-r-controlled", "The coldest season of the year", "We build snow forts in winter."),
    spellingWord("sister", "g2-r-controlled", "A girl who shares your parent", "My sister taught me a new game."),
    spellingWord("garden", "g2-r-controlled", "A place where flowers or vegetables grow", "Tomatoes are growing in our garden."),
    spellingWord("storm", "g2-r-controlled", "Weather with strong wind, rain, or snow", "The storm rattled the windows."),
    spellingWord("turn", "g2-r-controlled", "To change direction", "Turn left at the playground."),

    spellingWord("rabbit", "g2-two-syllable", "A furry animal with long ears", "The rabbit nibbled a crisp carrot."),
    spellingWord("planet", "g2-two-syllable", "A large world that travels around a star", "Earth is the planet we call home."),
    spellingWord("basket", "g2-two-syllable", "A container woven to carry things", "We packed lunch in the picnic basket."),
    spellingWord("napkin", "g2-two-syllable", "A small cloth or paper used while eating", "Please put a napkin beside each plate."),
    spellingWord("picnic", "g2-two-syllable", "A meal eaten outdoors", "Our family had a picnic by the lake."),

    spellingWord("jumping", "g2-endings", "Moving into the air from your feet", "The child was jumping over the rope."),
    spellingWord("boxes", "g2-endings", "More than one box", "We stacked the empty boxes neatly."),
    spellingWord("played", "g2-endings", "Took part in a game in the past", "We played soccer yesterday."),
    spellingWord("hoped", "g2-endings", "Wanted something to happen", "I hoped the rain would stop."),
    spellingWord("cries", "g2-endings", "Sheds tears or calls out", "The baby cries when she is hungry."),

    spellingWord("little", "g2-irregular", "Small in size", "A little snail crossed the path."),
    spellingWord("people", "g2-irregular", "More than one person", "Many people cheered for the team."),
    spellingWord("could", "g2-irregular", "Was able to", "I could hear music through the door."),
    spellingWord("would", "g2-irregular", "A word used for a possible choice", "I would choose the blue backpack."),
    spellingWord("again", "g2-irregular", "One more time", "Please read that funny page again."),
  ],
  3: [
    spellingWord("animal", "g3-multisyllable", "A living creature that is not a plant", "The zookeeper cared for every animal."),
    spellingWord("morning", "g3-multisyllable", "The first part of the day", "Birds began singing early in the morning."),
    spellingWord("kitchen", "g3-multisyllable", "The room where food is prepared", "A delicious smell drifted from the kitchen."),
    spellingWord("rainbow", "g3-multisyllable", "A colorful arc seen after rain", "A rainbow appeared above the hills."),
    spellingWord("together", "g3-multisyllable", "With one another in the same place", "We worked together to build the tower."),

    spellingWord("unhappy", "g3-prefixes", "Not feeling happy", "The soggy sandwich made him unhappy."),
    spellingWord("retell", "g3-prefixes", "To tell a story again", "Can you retell the story in your own words?"),
    spellingWord("preview", "g3-prefixes", "To look at something before the full version", "We watched a preview of the new show."),
    spellingWord("dislike", "g3-prefixes", "To not like something", "I dislike waking up before sunrise."),
    spellingWord("rewrite", "g3-prefixes", "To write something again", "I will rewrite the sentence more clearly."),

    spellingWord("careful", "g3-suffixes", "Taking care to avoid mistakes or danger", "Be careful when carrying the full glass."),
    spellingWord("helpful", "g3-suffixes", "Ready or able to help", "The map was helpful during our hike."),
    spellingWord("slowly", "g3-suffixes", "At a slow speed", "The turtle moved slowly across the path."),
    spellingWord("kindness", "g3-suffixes", "The quality of being kind", "Her kindness made the new student feel welcome."),
    spellingWord("movement", "g3-suffixes", "The act of changing position", "We noticed movement behind the curtain."),

    spellingWord("sitting", "g3-spelling-changes", "Resting on a seat", "The puppy was sitting near the door."),
    spellingWord("smiled", "g3-spelling-changes", "Made a happy expression in the past", "She smiled when she saw the surprise."),
    spellingWord("carried", "g3-spelling-changes", "Moved something while holding it", "We carried the groceries inside."),
    spellingWord("happiness", "g3-spelling-changes", "The feeling of being happy", "His face showed pure happiness."),
    spellingWord("running", "g3-spelling-changes", "Moving quickly on foot", "They were running around the track."),

    spellingWord("because", "g3-irregular", "A word that gives a reason", "We stayed inside because it was stormy."),
    spellingWord("friend", "g3-irregular", "A person you like and trust", "My friend helped me finish the puzzle."),
    spellingWord("answer", "g3-irregular", "A response to a question", "I wrote the answer under the question."),
    spellingWord("often", "g3-irregular", "Many times", "We often visit the park on Saturdays."),
    spellingWord("enough", "g3-irregular", "As much as is needed", "There is enough soup for everyone."),
  ],
  4: [
    spellingWord("calendar", "g4-multisyllable", "A chart that shows days, weeks, and months", "I marked the field trip on my calendar."),
    spellingWord("library", "g4-multisyllable", "A place where books can be read or borrowed", "I found a mystery novel at the library."),
    spellingWord("mountain", "g4-multisyllable", "A very high area of rocky land", "Snow covered the top of the mountain."),
    spellingWord("journey", "g4-multisyllable", "A trip from one place to another", "Our journey across the state took all day."),
    spellingWord("remember", "g4-multisyllable", "To bring something back to mind", "Remember to pack your water bottle."),

    spellingWord("separate", "g4-tricky", "To keep or move things apart", "Please separate the dark socks from the white ones."),
    spellingWord("surprise", "g4-tricky", "Something unexpected", "The secret party was a wonderful surprise."),
    spellingWord("favorite", "g4-tricky", "The one you like best", "Mango is my favorite fruit."),
    spellingWord("business", "g4-tricky", "Work involving goods or services", "The family opened a small business."),
    spellingWord("probably", "g4-tricky", "Very likely to happen", "It will probably rain this afternoon."),

    spellingWord("telephone", "g4-roots", "A device used to talk across a distance", "The telephone rang during dinner."),
    spellingWord("autograph", "g4-roots", "A person's written signature", "The author signed an autograph for me."),
    spellingWord("transport", "g4-roots", "To carry something from one place to another", "Trucks transport food across the country."),
    spellingWord("submarine", "g4-roots", "A vessel that travels underwater", "The submarine explored the deep ocean."),
    spellingWord("bicycle", "g4-roots", "A vehicle with two wheels", "She rode her bicycle to the park."),

    spellingWord("different", "g4-suffixes", "Not the same as something else", "Each shell had a different pattern."),
    spellingWord("curious", "g4-suffixes", "Eager to know or learn something", "The curious child asked many questions."),
    spellingWord("careless", "g4-suffixes", "Not giving enough attention", "A careless spill soaked the drawing."),
    spellingWord("comfortable", "g4-suffixes", "Feeling relaxed and at ease", "The soft chair was comfortable."),
    spellingWord("decision", "g4-suffixes", "A choice made after thinking", "Choosing a team name was a hard decision."),

    spellingWord("habitat", "g4-academic", "The natural home of a plant or animal", "The wetland is a habitat for many birds."),
    spellingWord("conservation", "g4-academic", "Protection of nature and resources", "Water conservation helps during a drought."),
    spellingWord("evidence", "g4-academic", "Information that helps prove something", "The scientist recorded evidence from the test."),
    spellingWord("describe", "g4-academic", "To tell what something is like", "Please describe the character in three sentences."),
    spellingWord("measure", "g4-academic", "To find the size or amount of something", "We used a ruler to measure the desk."),
  ],
  5: [
    spellingWord("photograph", "g5-greek-latin", "A picture made with a camera", "We took a photograph of the waterfall."),
    spellingWord("biology", "g5-greek-latin", "The study of living things", "Biology helps us understand plants and animals."),
    spellingWord("geography", "g5-greek-latin", "The study of places and Earth's features", "In geography we studied mountain ranges."),
    spellingWord("thermometer", "g5-greek-latin", "A tool that measures temperature", "The thermometer showed that the air was cold."),
    spellingWord("aquatic", "g5-greek-latin", "Living or growing in water", "The aquarium contains many aquatic plants."),

    spellingWord("permission", "g5-suffixes", "Approval to do something", "We asked permission before starting the project."),
    spellingWord("courageous", "g5-suffixes", "Very brave", "The courageous firefighter entered the building."),
    spellingWord("reliable", "g5-suffixes", "Able to be trusted", "Our reliable clock keeps accurate time."),
    spellingWord("curiosity", "g5-suffixes", "A strong desire to know something", "Her curiosity led to a clever experiment."),
    spellingWord("celebration", "g5-suffixes", "A joyful event for a special occasion", "The championship ended with a celebration."),

    spellingWord("environment", "g5-multisyllable", "The natural world and our surroundings", "Recycling helps protect the environment."),
    spellingWord("temperature", "g5-multisyllable", "A measure of how hot or cold something is", "The temperature dropped as evening arrived."),
    spellingWord("community", "g5-multisyllable", "People who live or work together in an area", "Our community planted trees in the park."),
    spellingWord("dictionary", "g5-multisyllable", "A tool that explains what words mean", "I checked the spelling in a dictionary."),
    spellingWord("adventure", "g5-multisyllable", "An exciting or unusual experience", "Exploring the cave became a great adventure."),

    spellingWord("knowledge", "g5-academic", "Facts and understanding gained by learning", "Reading builds knowledge about the world."),
    spellingWord("language", "g5-academic", "A system of words used to communicate", "She is learning a new language at school."),
    spellingWord("excellent", "g5-academic", "Extremely good", "Your science project showed excellent work."),
    spellingWord("communicate", "g5-academic", "To share information or ideas", "People communicate through speech and writing."),
    spellingWord("opportunity", "g5-academic", "A useful or favorable chance", "The contest was an opportunity to share her art."),

    spellingWord("beautiful", "g5-irregular", "Very pleasing to see or hear", "The sunset painted a beautiful orange sky."),
    spellingWord("necessary", "g5-irregular", "Needed or required", "Water is necessary for every living thing."),
    spellingWord("restaurant", "g5-irregular", "A place where meals are prepared and served", "The restaurant serves food from many cultures."),
    spellingWord("rhythm", "g5-irregular", "A repeated pattern of sound or movement", "The drummer kept a steady rhythm."),
    spellingWord("privilege", "g5-irregular", "A special right or advantage", "It is a privilege to represent our school."),
  ],
};

// The first 50 non-contraction selections from the 2024 Children's Picture
// Book Sight Words ranking, split into progressive frequency bands. "I" is
// omitted because a one-letter pronoun does not produce meaningful spelling
// practice. Rank preserves source provenance for later curriculum review.
// https://doi.org/10.1002/trtr.2309
const SIGHT_WORD_SKILL = {
  id: "sight-words",
  label: "Sight words",
  tip: "These words appear often in children's books. Map the sounds you know to letters, then learn any unexpected part by heart.",
};

const sightWord = (word, rank, clue, sentence) => ({
  word,
  rank,
  skill: SIGHT_WORD_SKILL.id,
  clue,
  sentence,
});

const SIGHT_WORDS = {
  1: [
    sightWord("the", 1, "Used before a particular person, place, or thing", "The moon glowed above us."),
    sightWord("and", 2, "Joins words or ideas together", "We packed apples and crackers."),
    sightWord("a", 3, "Used before one nonspecific thing", "A bird landed on the fence."),
    sightWord("to", 4, "Can show direction or introduce an action", "We walked to the playground."),
    sightWord("you", 6, "The person or people being spoken to", "You can choose the next book."),
    sightWord("in", 7, "Inside a place or thing", "The pencils are in the box."),
    sightWord("of", 8, "Shows belonging, connection, or amount", "We saw a flock of birds."),
    sightWord("it", 9, "Refers to a thing already mentioned", "The puppy found a ball and chased it."),
    sightWord("he", 10, "Refers to a boy or man", "He carried the basket carefully."),
    sightWord("is", 11, "A present-tense form of be", "The water is cold today."),
  ],
  2: [
    sightWord("was", 12, "A past-tense form of be", "The tiny seed was under the soil."),
    sightWord("for", 13, "Shows purpose or who receives something", "This card is for my teacher."),
    sightWord("on", 14, "Touching or supported by a surface", "The notebook is on the desk."),
    sightWord("that", 15, "Points to a particular thing or idea", "That cloud looks like a dragon."),
    sightWord("with", 16, "Together or accompanied by", "I baked muffins with my aunt."),
    sightWord("but", 17, "Connects ideas that contrast", "The path was muddy but safe."),
    sightWord("his", 18, "Belonging to a boy or man", "His backpack has a bright patch."),
    sightWord("all", 19, "The whole amount or every one", "All the lights turned on."),
    sightWord("they", 20, "Refers to more than one person or thing", "They planted flowers by the gate."),
    sightWord("my", 21, "Belonging to the speaker", "My shoes are beside the door."),
  ],
  3: [
    sightWord("so", 22, "Can show a result or mean very", "It rained, so we played indoors."),
    sightWord("be", 23, "To exist or have a state", "Please be ready before noon."),
    sightWord("she", 24, "Refers to a girl or woman", "She solved the riddle quickly."),
    sightWord("up", 25, "Toward a higher place", "We climbed up the steep hill."),
    sightWord("at", 26, "Shows a place, time, or target", "Meet me at the garden gate."),
    sightWord("are", 27, "A present-tense form of be", "The ripe peaches are sweet."),
    sightWord("one", 28, "The number before two", "Only one cookie remained."),
    sightWord("said", 29, "The past tense of say", "Nora said the answer clearly."),
    sightWord("what", 30, "A question word asking for information", "What made that funny sound?"),
    sightWord("this", 31, "Points to a nearby thing or idea", "This puzzle has one piece left."),
  ],
  4: [
    sightWord("when", 32, "A question or connecting word about time", "When will the concert begin?"),
    sightWord("we", 33, "The speaker together with other people", "We built a fort from blankets."),
    sightWord("me", 34, "Refers to the speaker as an object", "Please hand the marker to me."),
    sightWord("have", 35, "To own, hold, or experience something", "I have a story to share."),
    sightWord("as", 36, "Can compare things or describe a role", "The snow felt as soft as feathers."),
    sightWord("do", 37, "To perform an action", "Do your best on the challenge."),
    sightWord("like", 38, "To enjoy or find something pleasing", "I like stories about space."),
    sightWord("out", 39, "Away from the inside", "The rabbit hopped out of its burrow."),
    sightWord("can", 40, "To be able to do something", "You can finish one step at a time."),
    sightWord("her", 41, "Refers to or belongs to a girl or woman", "Her painting filled the whole page."),
  ],
  5: [
    sightWord("not", 42, "Makes a statement negative", "The answer is not on this page."),
    sightWord("then", 43, "At that time or next in order", "Mix the batter, then add the berries."),
    sightWord("your", 44, "Belonging to the person being spoken to", "Your idea could solve the problem."),
    sightWord("no", 45, "A negative answer or none", "There was no wind after sunset."),
    sightWord("there", 46, "At or in that place", "Set the empty basket over there."),
    sightWord("day", 47, "A period of twenty-four hours", "The first day of spring felt warm."),
    sightWord("just", 48, "Can mean exactly, fairly, or very recently", "The train has just arrived."),
    sightWord("see", 50, "To notice something with your eyes", "Can you see the nest in the tree?"),
    sightWord("little", 51, "Small in size or amount", "A little frog rested on the rock."),
    sightWord("time", 52, "How events are ordered and measured", "It is time to begin the experiment."),
  ],
};
