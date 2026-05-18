CREATE TABLE single (
    id INT,
    category TEXT,
    time TEXT,
    timeperiod INT,
    source TEXT,
    frequency INT,
    numbers INT,
    putone INT,
    puttwo INT,
    putthree INT,
    putoverthree INT,
    CONSTRAINT card_environment_single UNIQUE (id, category, time, timeperiod, source)
);

CREATE TABLE count (
    time TEXT,
    timeperiod INT,
    source TEXT,
    count INT,
    CONSTRAINT count_environment UNIQUE (time, timeperiod, source)
);

CREATE TABLE deck (
    name TEXT,
    time TEXT,
    timeperiod INT,
    source TEXT,
    count INT,
    CONSTRAINT card_environment_deck UNIQUE (name, time, timeperiod, source)
);
CREATE INDEX deck_time_index ON public.deck USING btree ("time");

CREATE TABLE tag (
    name TEXT,
    time TEXT,
    timeperiod INT,
    source TEXT,
    count INT,
    CONSTRAINT card_environment_tag UNIQUE (name, time, timeperiod, source)
);
CREATE INDEX tag_time_index ON public.tag USING btree ("time");

CREATE TABLE unknown_decks (
    deck TEXT,
    username TEXT,
    source TEXT,
    time TEXT,
    id SERIAL
);

CREATE TABLE matchup (
    source TEXT,
    decka TEXT,
    deckb TEXT,
    period TEXT,
    draw INT,
    lose INT,
    win INT,
    type TEXT default 'match',
    CONSTRAINT matchup_pk UNIQUE (source, decka, deckb, period, type)
);

CREATE TABLE startup (
    source TEXT,
    id INT,
    first BOOLEAN,
    period TEXT,
    draw INT,
    lose INT,
    win INT,
    CONSTRAINT card_period_startup UNIQUE (source, id, first, period)
);

CREATE TABLE catchup (
    source TEXT,
    id INT,
    opponent_deck TEXT,
    period TEXT,
    draw INT,
    lose INT,
    win INT,
    CONSTRAINT card_period_catchup UNIQUE (source, id, opponent_deck, period)
)
