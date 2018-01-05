#include "../lang.hh"
#include "../options/options.hh"
#include "../state.hh"
#include "etc.hh"
#include "models.hh"
#include <string>
#include <unordered_map>

// Maps country codes to English names
const static std::unordered_map<std::string, std::string> countries = {
    { "ad", "Andorra" }, { "ae", "United Arab Emirates" },
    { "af", "Afghanistan" }, { "ag", "Antigua and Barbuda" },
    { "ai", "Anguilla" }, { "al", "Albania" }, { "am", "Armenia" },
    { "ao", "Angola" }, { "aq", "Antarctica" }, { "ar", "Argentina" },
    { "as", "American Samoa" }, { "at", "Austria" }, { "au", "Australia" },
    { "aw", "Aruba" }, { "ax", "Aland Islands !" }, { "az", "Azerbaijan" },
    { "ba", "Bosnia and Herzegovina" }, { "bb", "Barbados" },
    { "bd", "Bangladesh" }, { "be", "Belgium" }, { "bf", "Burkina Faso" },
    { "bg", "Bulgaria" }, { "bh", "Bahrain" }, { "bi", "Burundi" },
    { "bj", "Benin" }, { "bl", "Saint Barthélemy" }, { "bm", "Bermuda" },
    { "bn", "Brunei Darussalam" }, { "bo", "Bolivia" }, { "bq", "Bonaire" },
    { "br", "Brazil" }, { "bs", "Bahamas" }, { "bt", "Bhutan" },
    { "bv", "Bouvet Island" }, { "bw", "Botswana" }, { "by", "Belarus" },
    { "bz", "Belize" }, { "ca", "Canada" }, { "cc", "Cocos (Keeling) Islands" },
    { "cd", "Congo" }, { "cf", "Central African Republic" }, { "cg", "Congo" },
    { "ch", "Switzerland" }, { "ci", "Cote d'Ivoire !" },
    { "ck", "Cook Islands" }, { "cl", "Chile" }, { "cm", "Cameroon" },
    { "cn", "China" }, { "co", "Colombia" }, { "cr", "Costa Rica" },
    { "cu", "Cuba" }, { "cv", "Cabo Verde" }, { "cw", "Curaçao" },
    { "cx", "Christmas Island" }, { "cy", "Cyprus" }, { "cz", "Czechia" },
    { "de", "Germany" }, { "dj", "Djibouti" }, { "dk", "Denmark" },
    { "dm", "Dominica" }, { "do", "Dominican Republic" }, { "dz", "Algeria" },
    { "ec", "Ecuador" }, { "ee", "Estonia" }, { "eg", "Egypt" },
    { "eh", "Western Sahara" }, { "er", "Eritrea" }, { "es", "Spain" },
    { "et", "Ethiopia" }, { "fi", "Finland" }, { "fj", "Fiji" },
    { "fk", "Falkland Islands (Malvinas)" }, { "fm", "Micronesia" },
    { "fo", "Faroe Islands" }, { "fr", "France" }, { "ga", "Gabon" },
    { "gb", "United Kingdom" }, { "gd", "Grenada" }, { "ge", "Georgia" },
    { "gf", "French Guiana" }, { "gg", "Guernsey" }, { "gh", "Ghana" },
    { "gi", "Gibraltar" }, { "gl", "Greenland" }, { "gm", "Gambia" },
    { "gn", "Guinea" }, { "gp", "Guadeloupe" }, { "gq", "Equatorial Guinea" },
    { "gr", "Greece" },
    { "gs", "South Georgia and the South Sandwich Islands" },
    { "gt", "Guatemala" }, { "gu", "Guam" }, { "gw", "Guinea-Bissau" },
    { "gy", "Guyana" }, { "hk", "Hong Kong" },
    { "hm", "Heard Island and McDonald Islands" }, { "hn", "Honduras" },
    { "hr", "Croatia" }, { "ht", "Haiti" }, { "hu", "Hungary" },
    { "id", "Indonesia" }, { "ie", "Ireland" }, { "il", "Israel" },
    { "im", "Isle of Man" }, { "in", "India" },
    { "io", "British Indian Ocean Territory" }, { "iq", "Iraq" },
    { "ir", "Iran" }, { "is", "Iceland" }, { "it", "Italy" },
    { "je", "Jersey" }, { "jm", "Jamaica" }, { "jo", "Jordan" },
    { "jp", "Japan" }, { "ke", "Kenya" }, { "kg", "Kyrgyzstan" },
    { "kh", "Cambodia" }, { "ki", "Kiribati" }, { "km", "Comoros" },
    { "kn", "Saint Kitts and Nevis" },
    { "kp", "Democratic People's Republic of Korea" },
    { "kr", "Republic of Korea" }, { "kw", "Kuwait" },
    { "ky", "Cayman Islands" }, { "kz", "Kazakhstan" },
    { "la", "Lao People's Democratic Republic" }, { "lb", "Lebanon" },
    { "lc", "Saint Lucia" }, { "li", "Liechtenstein" }, { "lk", "Sri Lanka" },
    { "lr", "Liberia" }, { "ls", "Lesotho" }, { "lt", "Lithuania" },
    { "lu", "Luxembourg" }, { "lv", "Latvia" }, { "ly", "Libya" },
    { "ma", "Morocco" }, { "mc", "Monaco" }, { "md", "Moldova" },
    { "me", "Montenegro" }, { "mf", "Saint Martin (French part)" },
    { "mg", "Madagascar" }, { "mh", "Marshall Islands" }, { "mk", "Macedonia" },
    { "ml", "Mali" }, { "mm", "Myanmar" }, { "mn", "Mongolia" },
    { "mo", "Macao" }, { "mp", "Northern Mariana Islands" },
    { "mq", "Martinique" }, { "mr", "Mauritania" }, { "ms", "Montserrat" },
    { "mt", "Malta" }, { "mu", "Mauritius" }, { "mv", "Maldives" },
    { "mw", "Malawi" }, { "mx", "Mexico" }, { "my", "Malaysia" },
    { "mz", "Mozambique" }, { "na", "Namibia" }, { "nc", "New Caledonia" },
    { "ne", "Niger" }, { "nf", "Norfolk Island" }, { "ng", "Nigeria" },
    { "ni", "Nicaragua" }, { "nl", "Netherlands" }, { "no", "Norway" },
    { "np", "Nepal" }, { "nr", "Nauru" }, { "nu", "Niue" },
    { "nz", "New Zealand" }, { "om", "Oman" }, { "pa", "Panama" },
    { "pe", "Peru" }, { "pf", "French Polynesia" },
    { "pg", "Papua New Guinea" }, { "ph", "Philippines" }, { "pk", "Pakistan" },
    { "pl", "Poland" }, { "pm", "Saint Pierre and Miquelon" },
    { "pn", "Pitcairn" }, { "pr", "Puerto Rico" }, { "ps", "Palestine" },
    { "pt", "Portugal" }, { "pw", "Palau" }, { "py", "Paraguay" },
    { "qa", "Qatar" }, { "re", "Reunion !" }, { "ro", "Romania" },
    { "rs", "Serbia" }, { "ru", "Russian Federation" }, { "rw", "Rwanda" },
    { "sa", "Saudi Arabia" }, { "sb", "Solomon Islands" },
    { "sc", "Seychelles" }, { "sd", "Sudan" }, { "se", "Sweden" },
    { "sg", "Singapore" }, { "sh", "Saint Helena" }, { "si", "Slovenia" },
    { "sj", "Svalbard and Jan Mayen" }, { "sk", "Slovakia" },
    { "sl", "Sierra Leone" }, { "sm", "San Marino" }, { "sn", "Senegal" },
    { "so", "Somalia" }, { "sr", "Suriname" }, { "ss", "South Sudan" },
    { "st", "Sao Tome and Principe" }, { "sv", "El Salvador" },
    { "sx", "Sint Maarten (Dutch part)" }, { "sy", "Syrian Arab Republic" },
    { "sz", "Swaziland" }, { "tc", "Turks and Caicos Islands" },
    { "td", "Chad" }, { "tf", "French Southern Territories" }, { "tg", "Togo" },
    { "th", "Thailand" }, { "tj", "Tajikistan" }, { "tk", "Tokelau" },
    { "tl", "Timor-Leste" }, { "tm", "Turkmenistan" }, { "tn", "Tunisia" },
    { "to", "Tonga" }, { "tr", "Turkey" }, { "tt", "Trinidad and Tobago" },
    { "tv", "Tuvalu" }, { "tw", "Taiwan" }, { "tz", "Tanzania" },
    { "ua", "Ukraine" }, { "ug", "Uganda" },
    { "um", "United States Minor Outlying Islands" },
    { "us", "United States of America" }, { "uy", "Uruguay" },
    { "uz", "Uzbekistan" }, { "va", "Holy See" },
    { "vc", "Saint Vincent and the Grenadines" }, { "ve", "Venezuela" },
    { "vg", "British Virgin Islands" }, { "vi", "U.S. Virgin Islands" },
    { "vn", "Viet Nam" }, { "vu", "Vanuatu" }, { "wf", "Wallis and Futuna" },
    { "ws", "Samoa" }, { "ye", "Yemen" }, { "yt", "Mayotte" },
    { "za", "South Africa" }, { "zm", "Zambia" }, { "zw", "Zimbabwe" }
};

Node Post::render_header()
{
    Node n = { "header", { { "class", "spaced" } } };
    n.children.reserve(4);

    // TODO: Check if staff, and render moderator checkbox

    if (id == op && !page->thread && page->board == "all") {
        n.children.push_back(
            { "b", { { "class", "board" } }, '/' + board + '/' });
    }
    if (sticky) {
        n.children.push_back({
            "svg",
            {
                { "xmlns", "http://www.w3.org/2000/svg" }, { "width", "8" },
                { "height", "8" }, { "viewBox", "0 0 8 8" },
            },
            R"'(<path d="M1.34 0a.5.5 0 0 0 .16 1h.5v2h-1c-.55 0-1 .45-1 1h3v3l.44 1 .56-1v-3h3c0-.55-.45-1-1-1h-1v-2h.5a.5.5 0 1 0 0-1h-4a.5.5 0 0 0-.09 0 .5.5 0 0 0-.06 0z" />)'",
        });
    }
    if (locked) {
        n.children.push_back({
            "svg",
            {
                { "xmlns", "http://www.w3.org/2000/svg" }, { "width", "8" },
                { "height", "8" }, { "viewBox", "0 0 8 8" },
            },
            R"'(<path d="M3 0c-1.1 0-2 .9-2 2v1h-1v4h6v-4h-1v-1c0-1.1-.9-2-2-2zm0 1c.56 0 1 .44 1 1v1h-2v-1c0-.56.44-1 1-1z" transform="translate(1)" />)'",
        });
    }

    if (id == op) {
        auto const& subject = threads->at(id).subject;
        std::string s;
        s.reserve(subject.size() + 8); // +2 unicode chars
        s = "「" + subject + "」";
        n.children.push_back({ "h3", s, true });
    }
    n.children.push_back(render_name());
    if (flag) {
        n.children.push_back({
            "img",
            {
                { "class", "flag" },
                { "src", "/assets/flags/" + *flag + ".svg" },
                {
                    "title",
                    countries.count(*flag) ? countries.at(*flag) : *flag,
                },
            },
        });
    }
    n.children.push_back(render_time());

    const auto id_str = std::to_string(id);
    std::string url = "#p" + id_str;
    if (!page->thread && !page->catalog) {
        url = "/all/" + id_str + "?last=100" + url;
    }
    n.children.push_back({
        "nav", {},
        {
            {
                "a",
                {
                    { "href", url },
                },
                "No.",
            },
            {
                "a",
                {
                    { "class", "quote" }, { "href", url },
                },
                id_str,
            },
        },
    });

    if (id == op && !page->thread && !page->catalog) {
        n.children.push_back(
            { "span", {}, brunhild::Children({ render_expand_link(board, id),
                              render_last_100_link(board, id) }) });
    }

    n.children.push_back({ "a", { { "class", "control" } },
        R"'(<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8" viewBox="0 0 8 8"><path d="M1.5 0l-1.5 1.5 4 4 4-4-1.5-1.5-2.5 2.5-2.5-2.5z" transform="translate(0 1)" /></svg>)'" });

    n.stringify_subtree();
    return n;
}

Node Post::render_name()
{
    Node n("b", { { "class", "name spaced" } });
    if (sage) {
        n.attrs["class"] += " sage";
    }

    if (options->anonymise) {
        n.children = { Node("span", lang->posts.at("anon")) };
        return n;
    }

    if (name || !trip) {
        n.children.push_back(name ? Node("span", *name, true)
                                  : Node("span", lang->posts.at("anon")));
    }
    if (trip) {
        n.children.push_back({ "code", "!" + *trip, true });
    }
    if (poster_id) {
        n.children.push_back({ "span", *poster_id, true });
    }
    if (auth) {
        n.attrs["class"] += " admin";
        n.children.push_back({ "span", "## " + lang->posts.at(*auth) });
    }
    if (post_ids->mine.count(id)) {
        n.children.push_back({ "i", lang->posts.at("you") });
    }

    return n;
}

Node Post::render_time()
{
    using std::setw;

    auto then = std::localtime(&time);

    // Renders classic absolute timestamp
    std::ostringstream abs;
    abs << std::setfill('0') << setw(2) << then->tm_mday << ' '
        << lang->calendar[then->tm_mon] << ' ' << 1900 + then->tm_year << " ("
        << lang->week[then->tm_wday] << ") " << setw(2) << then->tm_hour << ':'
        << setw(2) << then->tm_min;

    const auto rel = relative_time(time);

    return Node("time",
        { { "title", options->relative_time ? abs.str() : rel } },
        options->relative_time ? rel : abs.str());
}
