#!/usr/bin/env python3

import csv
from decimal import Decimal
from datetime import datetime
import re

CATEGORY_MAPPING = {
    "Alimentation": "Groceries",
    "Banque et assurances": "Finance",
    "Juridique et administratif": "Finance",
    "Logement - maison": "Housing",
    "Loisirs et vacances": "Travel",
    "Revenus et rentrees d'argent": "Income",
    "Sante": "Health",
    "Shopping et services": "Lifestyle",
    "Transports": "Transport",
    "A categoriser - rentree d'argent": "Uncategorized",
    "A categoriser - sortie d'argent": "Uncategorized",
}


SUBCATEGORY_MAPPING = {
    "Restaurant": "Restaurants",
    "Hyper/supermarche": "Groceries",
    "Alimentation - autre": "Groceries",
}


REGEX_RULES = [
    (re.compile(r"\b(sas aih|SAS ST LA GARONNE|pea yomoni|papa|olivier)\b", re.I), "Finance"),
    (re.compile(r"\b(martin jerome william|martin guyard|guyard martin|N26|fortuneo|retrait d)\b", re.I), "Transfer"),
    (re.compile(r"\b(datadog|france travail)\b", re.I), "Salary"),
    (re.compile(r"\b(dgfip)\b", re.I), "Taxes"),
    (re.compile(r"\b(adn gestion)\b", re.I), "Housing"),
    (re.compile(r"\b(juliette)\b", re.I), "Lifestyle"),
    (re.compile(r"\b(henner)\b", re.I), "Health"),
    (re.compile(r"\b(santander)\b", re.I), "Finance"),
    (re.compile(r"\b(telecom)\b", re.I), "Housing"),
    # (re.compile(r"\b(monoprix|carrefour|leclerc|auchan|franprix|lidl|intermarch[eé])\b", re.I), "Groceries"),
    # (re.compile(r"\b(uber\s?eats|deliveroo|resto|restaurant|brasserie|bistro)\b", re.I), "Restaurants"),
    # (re.compile(r"\b(sncf|ratp|uber|bolt|free\s*now|tgv)\b", re.I), "Transport"),
    # (re.compile(r"\b(amazon|fnac|ikea|zalando)\b", re.I), "Lifestyle"),
    # (re.compile(r"\b(pharmacie|docteur|laboratoire|hopital|hôpital)\b", re.I), "Health"),
    # (re.compile(r"\b(edf|engie|veolia|suez)\b", re.I), "Housing"),
]

def parse_amount(value: str) -> Decimal:
    if not value:
        return Decimal("0")

    return Decimal(value.replace(",", "."))


def convert_date(value: str) -> str:
    return datetime.strptime(value, "%d/%m/%Y").strftime("%Y-%m-%d")


with open("transactions.csv", encoding="utf-8") as src, \
     open("items.csv", "w", newline="", encoding="utf-8") as dst:

    reader = csv.DictReader(src, delimiter=";")

    writer = csv.DictWriter(
        dst,
        fieldnames=[
            "type",
            "name",
            "amount",
            "date",
            "category",
        ],
    )

    items = []

    for row in reader:
        debit = parse_amount(row["Debit"])
        credit = parse_amount(row["Credit"])

        amount = debit if debit != 0 else credit

        if amount == 0:
            continue

        item_type = "EXPENSE" if amount < 0 else "INCOME"

        raw_category = CATEGORY_MAPPING.get(
            row["Categorie"].strip(),
            "Uncategorized",
        )

        subcategory = SUBCATEGORY_MAPPING.get(
            row["Sous categorie"].strip(),
            None,
        )

        if subcategory:
            category = subcategory
        else:
            category = raw_category

        # fallback regex ONLY if still uncategorized
        if category == "Uncategorized":
            libelle = row["Libelle simplifie"].strip()

            for pattern, mapped_category in REGEX_RULES:
                if pattern.search(libelle.lower()):
                    category = mapped_category
                    break

        date = convert_date(row["Date de comptabilisation"])

        items.append({
            "type": item_type,
            "name": row["Libelle simplifie"].strip(),
            "amount": str(abs(amount)),
            "date": date,
            "category": category,
        })

    # latest first
    items.sort(key=lambda item: item["date"], reverse=True)

    for item in items:
        writer.writerow(item)

print("Generated items.csv")
