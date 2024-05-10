--
-- PostgreSQL database dump
--

-- Dumped from database version 12.18 (Ubuntu 12.18-0ubuntu0.20.04.1)
-- Dumped by pg_dump version 12.18 (Ubuntu 12.18-0ubuntu0.20.04.1)

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
-- Name: mc_iam_mapping_workspace_projects; Type: TABLE; Schema: public; Owner: mciam
--

CREATE TABLE public.mc_iam_mapping_workspace_projects (
    id uuid NOT NULL,
    workspace_id text NOT NULL,
    project_id text NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


ALTER TABLE public.mc_iam_mapping_workspace_projects OWNER TO mciam;

--
-- Name: mc_iam_mapping_workspace_user_roles; Type: TABLE; Schema: public; Owner: mciam
--

CREATE TABLE public.mc_iam_mapping_workspace_user_roles (
    id uuid NOT NULL,
    workspace_id text NOT NULL,
    role_name text NOT NULL,
    user_id text NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


ALTER TABLE public.mc_iam_mapping_workspace_user_roles OWNER TO mciam;

--
-- Name: mc_iam_projects; Type: TABLE; Schema: public; Owner: mciam
--

CREATE TABLE public.mc_iam_projects (
    id uuid NOT NULL,
    project_id text NOT NULL,
    name text NOT NULL,
    description text,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


ALTER TABLE public.mc_iam_projects OWNER TO mciam;

--
-- Name: mc_iam_roletypes; Type: TABLE; Schema: public; Owner: mciam
--

CREATE TABLE public.mc_iam_roletypes (
    id uuid NOT NULL,
    type text NOT NULL,
    role_id text NOT NULL,
    role_name text NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


ALTER TABLE public.mc_iam_roletypes OWNER TO mciam;

--
-- Name: mc_iam_workspaces; Type: TABLE; Schema: public; Owner: mciam
--

CREATE TABLE public.mc_iam_workspaces (
    id uuid NOT NULL,
    workspace_id text NOT NULL,
    name text NOT NULL,
    description text,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


ALTER TABLE public.mc_iam_workspaces OWNER TO mciam;

--
-- Name: schema_migration; Type: TABLE; Schema: public; Owner: mciam
--

CREATE TABLE public.schema_migration (
    version character varying(14) NOT NULL
);


ALTER TABLE public.schema_migration OWNER TO mciam;

--
-- Name: mc_iam_mapping_workspace_projects mc_iam_mapping_workspace_projects_pkey; Type: CONSTRAINT; Schema: public; Owner: mciam
--

ALTER TABLE ONLY public.mc_iam_mapping_workspace_projects
    ADD CONSTRAINT mc_iam_mapping_workspace_projects_pkey PRIMARY KEY (id);


--
-- Name: mc_iam_mapping_workspace_user_roles mc_iam_mapping_workspace_user_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: mciam
--

ALTER TABLE ONLY public.mc_iam_mapping_workspace_user_roles
    ADD CONSTRAINT mc_iam_mapping_workspace_user_roles_pkey PRIMARY KEY (id);


--
-- Name: mc_iam_projects mc_iam_projects_pkey; Type: CONSTRAINT; Schema: public; Owner: mciam
--

ALTER TABLE ONLY public.mc_iam_projects
    ADD CONSTRAINT mc_iam_projects_pkey PRIMARY KEY (id);


--
-- Name: mc_iam_roletypes mc_iam_roletypes_pkey; Type: CONSTRAINT; Schema: public; Owner: mciam
--

ALTER TABLE ONLY public.mc_iam_roletypes
    ADD CONSTRAINT mc_iam_roletypes_pkey PRIMARY KEY (id);


--
-- Name: mc_iam_workspaces mc_iam_workspaces_pkey; Type: CONSTRAINT; Schema: public; Owner: mciam
--

ALTER TABLE ONLY public.mc_iam_workspaces
    ADD CONSTRAINT mc_iam_workspaces_pkey PRIMARY KEY (id);


--
-- Name: schema_migration schema_migration_pkey; Type: CONSTRAINT; Schema: public; Owner: mciam
--

ALTER TABLE ONLY public.schema_migration
    ADD CONSTRAINT schema_migration_pkey PRIMARY KEY (version);


--
-- Name: schema_migration_version_idx; Type: INDEX; Schema: public; Owner: mciam
--

CREATE UNIQUE INDEX schema_migration_version_idx ON public.schema_migration USING btree (version);


--
-- PostgreSQL database dump complete
--

