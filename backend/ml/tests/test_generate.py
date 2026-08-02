from ticketlens_ml.generate import examples_to_frame, generate_examples
from ticketlens_ml.taxonomy import CATEGORIES, PRIORITIES


def test_generate_covers_all_categories_and_schema():
    examples = generate_examples(per_category=5, seed=1)
    df = examples_to_frame(examples)
    assert set(df["category"]) == set(CATEGORIES)
    assert set(df["priority"]).issubset(set(PRIORITIES))
    assert list(df.columns) == [
        "subject",
        "body",
        "category",
        "priority",
        "source",
        "lang",
        "generated_by",
    ]
    assert (df["lang"] == "en").all()
    assert (df["source"] == "synthetic").all()
    assert len(df) == 5 * len(CATEGORIES)


def test_generate_is_deterministic():
    a = examples_to_frame(generate_examples(per_category=3, seed=42))
    b = examples_to_frame(generate_examples(per_category=3, seed=42))
    assert a.equals(b)
