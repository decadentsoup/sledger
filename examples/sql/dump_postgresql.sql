--
-- PostgreSQL database dump
--

-- Dumped from database version 15.3
-- Dumped by pg_dump version 15.3

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: account; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.account (
    username text NOT NULL,
    password text NOT NULL
);


ALTER TABLE public.account OWNER TO postgres;

--
-- Name: post; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.post (
    title text NOT NULL,
    body text NOT NULL
);


ALTER TABLE public.post OWNER TO postgres;

--
-- Name: sledger; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.sledger (
    index bigint NOT NULL,
    forward text NOT NULL,
    backward text NOT NULL,
    "timestamp" timestamp without time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.sledger OWNER TO postgres;

--
-- Name: sledger_version; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.sledger_version (
    sledger_version text
);


ALTER TABLE public.sledger_version OWNER TO postgres;

--
-- Data for Name: account; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.account (username, password) FROM stdin;
system	sandwich
\.


--
-- Data for Name: post; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.post (title, body) FROM stdin;
\.


--
-- Data for Name: sledger; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.sledger (index, forward, backward, "timestamp") FROM stdin;
0	CREATE TABLE account (username text NOT NULL, password text NOT NULL)	DROP TABLE account	(timestamp)
1	CREATE TABLE post (title text NOT NULL, body text NOT NULL)	DROP TABLE post	(timestamp)
2	INSERT INTO account (username, password) VALUES ('system', 'sandwich')	DELETE FROM account WHERE username = 'system'	(timestamp)
\.


--
-- Data for Name: sledger_version; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.sledger_version (sledger_version) FROM stdin;
a45a9821-8e0d-4126-8d99-0543e7f1f8f7
\.


--
-- PostgreSQL database dump complete
--

