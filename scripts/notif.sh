#!/bin/bash

export PROJECT=tb

generate_post_data()
{
  cat <<EOF
{
  "cards":[
    {
      "header":{
        "title":"${DEPLOY_MESSAGE}",
        "subtitle":"${PROJECT}",
        "imageUrl":"${CARD_IMAGE}",
        "imageStyle":"IMAGE"
      },
      "sections":[
        {
          "widgets":[
            {
              "keyValue":{
                "content":"Branch: ${BRANCH}"
              }
            },
            {
              "keyValue":{
                "content":"Commit SHA: ${REVISION}"
              }
            },
            {
              "keyValue":{
                "content":"User: ${SHIPIT_USER}"
              }
            }
          ]
        },
        {
          "widgets":[
            {
              "buttons":[
                {
                  "textButton":{
                    "text":"DETAILS",
                    "onClick":{
                      "openLink":{
                        "url":"${SHIPIT_LINK}"
                      }
                    }
                  }
                },
                {
                  "textButton":{
                    "text":"DIFF",
                    "onClick":{
                      "openLink":{
                        "url":"${DIFF_LINK}"
                      }
                    }
                  }
                }
              ]
            }
          ]
        }
      ]
    }
  ]
}
EOF
}
