import datetime

import mariadb
import openai

DB_HOST = 'localhost'
DB_USER = 'forum2'
DB_PASSWORD = '123'
DB_NAME = 'forum2'
GPT_KEY = ""
GPT_CONSIGNES = """
Tu es fan de sports mécaniques et de voiture.
Tu es sur un forum nommé "Vroom Forum" qui parle de sports mécaniques.
Tu as des opinions tranchées et tu utilises le second degré.
"""
GPT_NEW_TOPIC = """
Lance un nouveau sujet de discussion (débat) sur quelque chose qui fait partie des sports mécaniques.
Commence ta réponse par un court titre. Ensuite, met un point et commence ton développement.
"""
BOT_ID_USER = 2
NB_NEW_TOPICS = 3
NB_NEW_MESSAGES = 3


def new_topic(db, cursor, message, title):
    cursor.execute("INSERT INTO topics (title, id_cat) VALUES (%s, %s)", (title, 1))
    db.commit()
    id_topic = cursor.lastrowid

    time = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    cursor.execute("INSERT INTO messages (content, id_user, date_created, id_topic) VALUES (%s, %s, %s, %s)",
	    (message, BOT_ID_USER, time, id_topic))
    db.commit()

    return id_topic


def new_message(db, cursor, message, id_topic):
    time = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    cursor.execute("INSERT INTO messages (content, id_user, date_created, id_topic) VALUES (%s, %s, %s, %s)",
	    (message, BOT_ID_USER, time, id_topic))
    db.commit()


db = mariadb.connect(
    host=DB_HOST,
    user=DB_USER,
    password=DB_PASSWORD,
    database=DB_NAME
)
cursor = db.cursor()

openai.api_key = GPT_KEY

for t in range(NB_NEW_TOPICS):
    reponse = openai.ChatCompletion.create(model="gpt-3.5-turbo", messages=[
                            {"role": "system", "content": GPT_CONSIGNES},
                            {"role": "user", "content": GPT_NEW_TOPIC}], temperature=1.1)
    topic_content = reponse["choices"][0]["message"]["content"].split(".", 1)
    id_topic = new_topic(db, cursor, topic_content[1], topic_content[0][:50])
    print(f"topics: {t+1}/{NB_NEW_TOPICS}")
    for m in range(NB_NEW_MESSAGES):
        reponse = openai.ChatCompletion.create(model="gpt-3.5-turbo", messages=[
                            {"role": "system", "content": GPT_CONSIGNES},
                            {"role": "user", "content": topic_content[1]}], temperature=1.1)
        message_content = reponse["choices"][0]["message"]["content"]
        new_message(db, cursor, message_content, id_topic)

cursor.close()
db.close()

